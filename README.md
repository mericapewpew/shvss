# shvss
Self Hosted Video Subscription Server

## Description
A server that displays youtube and odysee videos in chronological order based on what you add to the subs div.

## Installation
* install build requirements
```shell
# arch based
pacman -Sy go git
# debian based
apt install golang-go git
```
* install (systemd based)
```shell
git clone --depth 1 https://github.com/mericapewpew/shvss.git
cd shvss/
bash installer.sh install
```

## Adding subs
where to find required formats

### YouTube

---
* on a video, click share
* click embed
* click user picture in embed window

it will be the UC*************  in the url

![](.gitassets/youtube.png)

### Odysee

---
in the url, including claim id

![](.gitassets/odysee.png)

### Rumble (broken)

---
rumble is implemented, but the embed_ID and video_ID are different, parsing each page to extract the embed id is slow

"embedUrl": "https://rumble.com/embed/{embed_ID}/" is in the page as a json variable 