import os
import time
import signal
import threading
import requests
from dataclasses import dataclass
from typing import Dict, List, Optional
from datetime import datetime, timezone

import cv2
from ultralytics import YOLO


DOG_CLASS_ID = 16
PERSON_CLASS_ID = 0

CLASS_NAMES = {
    DOG_CLASS_ID: "dog",
    PERSON_CLASS_ID: "person"
}

@dataclass
class Config:
    streams_api: str
    rtsp_url_base: str
    model_path: str
    confidence: float
    frame_stride: int
    reconnect_delay_seconds: float
    stream_poll_seconds: float
    log_every_seconds: float
    detections_api: str


def load_config() -> Config:
    return Config(
        streams_api=os.getenv(
            "STREAMS_API",
            "http://host.docker.internal:8000/api/streams",
        ),
        rtsp_url_base=os.getenv("RTSP_URL_BASE", "rtsp://mediamtx:8554/"),
        model_path=os.getenv("MODEL_PATH", "yolov8n.pt"),
        confidence=float(os.getenv("CONFIDENCE", "0.45")),
        frame_stride=int(os.getenv("FRAME_STRIDE", "5")),
        reconnect_delay_seconds=float(
            os.getenv("RECONNECT_DELAY_SECONDS", "3")),
        stream_poll_seconds=float(os.getenv("STREAM_POLL_SECONDS", "5")),
        log_every_seconds=float(os.getenv("LOG_EVERY_SECONDS", "5")),
        detections_api=os.getenv(
            "DETECTIONS_API",
            "http://host.docker.internal:8000/api/detections",
        ),
    )


class StreamWorker:
    def __init__(self, stream_name: str, config: Config, model: YOLO, model_lock: threading.Lock):
        self.stream_name = stream_name
        self.config = config
        self.model = model
        self.stop_event = threading.Event()
        self.thread = threading.Thread(
            target=self.run,
            name=f"detector-{stream_name}",
            daemon=True,
        )
        self.model_lock = model_lock
        self.session = requests.Session()

    def start(self):
        print(f"> [{self.stream_name}] starting worker...", flush=True)
        self.thread.start()

    def stop(self):
        print(f"> [{self.stream_name}] stopping worker...", flush=True)
        self.stop_event.set()

    def join(self, timeout: Optional[float] = None):
        self.thread.join(timeout=timeout)

    def rtsp_url(self):
        return (
            self.config.rtsp_url_base.rstrip("/")
            + "/"
            + self.stream_name.lstrip("/")
        )

    def open_stream(self):
        url = self.rtsp_url()
        print(f"> [{self.stream_name}] opening RTSP stream: {
              url}...", flush=True)

        cap = cv2.VideoCapture(url, cv2.CAP_FFMPEG)

        if not cap.isOpened():
            cap.release()
            return None

        return cap

    def run(self):
        while not self.stop_event.is_set():
            cap = self.open_stream()

            if cap is None:
                print(
                    f"> [{self.stream_name}] could not open stream; "
                    f"retrying in {self.config.reconnect_delay_seconds}s...",
                    flush=True,
                )
                self.stop_event.wait(self.config.reconnect_delay_seconds)
                continue

            try:
                self.process_stream(cap)
            finally:
                cap.release()

            if not self.stop_event.is_set():
                print(
                    f"> [{self.stream_name}] stream ended; reconnecting in "
                    f"{self.config.reconnect_delay_seconds}s...",
                    flush=True,
                )
                self.stop_event.wait(self.config.reconnect_delay_seconds)

        print(f"> [{self.stream_name}] worker stopped", flush=True)

    def process_stream(self, cap):
        frame_count = 0
        last_status_log = 0.0
        last_detection_log = 0.0

        while not self.stop_event.is_set():
            ok, frame = cap.read()

            if not ok or frame is None:
                print(f"[{self.stream_name}] failed to read frame", flush=True)
                return

            frame_count += 1

            if frame_count % self.config.frame_stride != 0:
                continue

            now = time.time()

            if now - last_status_log >= self.config.log_every_seconds:
                print(
                    f"[{self.stream_name}] processed {frame_count} frames",
                    flush=True,
                )
                last_status_log = now

            detections = self.detect_objects(frame)

            dog_detected = any(d["class"] == "dog" for d in detections)
            person_detected = any(d["class"] == "person" for d in detections)

            if (dog_detected or person_detected) and now - last_detection_log >= 1:
                print(
                    f"[{self.stream_name}] OBJECTS DETECTED: "
                    f"dog_detected={dog_detected} "
                    f"person_detected={person_detected}",
                    flush=True,
                )

                self.publish_detection_events(dog_detected, person_detected)
                last_detection_log = now

    def detect_objects(self, frame):
        with self.model_lock:
            results = self.model.predict(
                source=frame,
                conf=self.config.confidence,
                classes=[PERSON_CLASS_ID, DOG_CLASS_ID],
                verbose=False,
            )

        detections = []

        for result in results:
            if result.boxes is None:
                continue

            for box in result.boxes:
                class_id = int(box.cls[0])
                confidence = float(box.conf[0])
                x1, y1, x2, y2 = box.xyxy[0].tolist()

                detections.append(
                    {
                        "class": CLASS_NAMES.get(class_id, "unknown"),
                        "class_id": class_id,
                        "confidence": round(confidence, 3),
                        "box": {
                            "x1": int(x1),
                            "y1": int(y1),
                            "x2": int(x2),
                            "y2": int(y2),
                        },
                    }
                )

        return detections

    def publish_detection_events(self, dog_detected: bool, person_detected: bool):
        payload = {
            "stream": self.stream_name,
            "dog_detected": dog_detected,
            "person_detected": person_detected,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }

        try:
            resp = self.session.post(
                self.config.detections_api,
                json=payload,
                timeout=2,
            )
            resp.raise_for_status()
        except requests.RequestException as err:
            print(
                f"[{self.stream_name}] failed to publish detection event: {err}",
                flush=True,
            )


class DetectorSupervisor:
    def __init__(self, config: Config):
        self.config = config
        self.running = True
        self.model = YOLO(config.model_path)
        self.model_lock = threading.Lock()
        self.workers: Dict[str, StreamWorker] = {}

    def stop(self, *_args):
        print("> Shutdown signal received...", flush=True)
        self.running = False

        for worker in list(self.workers.values()):
            worker.stop()

    def get_live_streams(self) -> List[str]:
        try:
            resp = requests.get(self.config.streams_api, timeout=3)
            resp.raise_for_status()

            data = resp.json()
            streams = data.get("streams", [])

            if not isinstance(streams, list):
                print(f"invalid streams response: {data}", flush=True)
                return []

            return [stream for stream in streams if isinstance(stream, str) and stream]

        except requests.RequestException as err:
            print(f"failed to fetch live streams: {err}", flush=True)
            return []

    def reconcile_workers(self, live_streams: List[str]):
        live = set(live_streams)
        current = set(self.workers.keys())

        streams_to_start = live - current
        streams_to_stop = current - live

        for stream_name in sorted(streams_to_stop):
            worker = self.workers.pop(stream_name)
            worker.stop()
            worker.join(timeout=2)

        for stream_name in sorted(streams_to_start):
            worker = StreamWorker(
                stream_name=stream_name,
                config=self.config,
                model=self.model,
                model_lock=self.model_lock,
            )
            self.workers[stream_name] = worker
            worker.start()

    def run(self):
        print("[ Object Detection Started... ]", flush=True)
        print(f"[ Streams API: {self.config.streams_api} ]", flush=True)
        print(f"[ RTSP URL base: {self.config.rtsp_url_base} ]", flush=True)
        print(f"[ Model: {self.config.model_path} ]", flush=True)
        print(f"[ Confidence threshold: {
              self.config.confidence} ]", flush=True)
        print(f"[ Frame stride: {self.config.frame_stride} ]", flush=True)
        print(f"[ Stream poll seconds: {
              self.config.stream_poll_seconds} ]", flush=True)

        while self.running:
            live_streams = self.get_live_streams()

            if live_streams:
                print(f"> Live Streams Found: {live_streams}", flush=True)
            else:
                print("no live streams found", flush=True)

            self.reconcile_workers(live_streams)

            time.sleep(self.config.stream_poll_seconds)

        print("stopping stream workers", flush=True)

        for worker in list(self.workers.values()):
            worker.stop()

        for worker in list(self.workers.values()):
            worker.join(timeout=5)

        self.workers.clear()

        print("detector stopped", flush=True)


def main():
    config = load_config()
    supervisor = DetectorSupervisor(config)

    signal.signal(signal.SIGINT, supervisor.stop)
    signal.signal(signal.SIGTERM, supervisor.stop)

    supervisor.run()


if __name__ == "__main__":
    main()
