<p align="center">
  <img src="" alt="Sentry Icon (soon)" width="128" />
</p>

<h1 align="center">Sentry</h1>

<p align="center">
  A hub-and-spoke networked video surveillance system written in Go
</p>

## Compatibility

- Sentry Hub is compatible with both Linux and MacOS machines running `Docker`
- Sentry Node is designed for the Raspberry Pi 5 running Debian Bookworm utilizing `rpicam` and `ffmpeg`

## How to Build

This project includes a Makefile for easy building:

- Create an `.env` file in the root directory that includes `HUB_DEVICE` and `NODE_DEVICE`
- Run `make hub`, `make node`, or `make all`
- **You will need to change the hub command for a Hub Linux binary**

## How to Run

#### Hub

- For testing or development:
  - `./sentry-hub`
- For production:
  - `./sentry-hub --deploy`

**_The `--deploy` flag gives you an opportunity to configure a Cloudflare Tunnel for your `Hub`_**
**_When prompted for a hostname, use a subdomain for a domain that exists in your Cloudflare account_**
**_Example: sentry.your-domain-here.com_**

**_It's a good idea to configure a `zero trust application` as well_**

#### Node

- For testing or development:
  - `./sentry-node`
- For production:
  - `./sentry-node --deploy`

**_The `--deploy` flag will run the program as a `systemd` service_**

## Features and Roadmap

- [x] Real-time 720p 30fps video streaming
- [x] Web front end for viewing and controlling camera nodes
- [x] Object detection events for dogs and people
- [x] Camera node aliases for easy stream identification
- [x] Easy Cloudflare Tunnel deployment
- [ ] Notification service driven by object detection events
- [ ] Additional service configuration
- [ ] Stored recordings using local or cloud storage
- [ ] Node optimizations for running on a Pi Zero 2w

## How It Works

When a Node starts, it automatically searches for a Sentry Hub broadcasting on the local network.
Once the necessary endpoints are discovered and connections are initialized, the Node begins publishing video to the Hub.
In the `Sentry Command Center` (working title), streams and object detection events are viewable.
The web front end can be found at: `http://<LAN IP>:8000/watch`.
Here the user can also turn on / off individual streams and create node `aliases`.

## Why?

This project will be a personal replacement for big name, indoor camera products.
The goal is to be cheaper and more private than similar products while also having a competitive set of features.
