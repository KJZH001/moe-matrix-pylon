# matrix-pylon

Matrix-Pylon is a universal bridge for [Matrix](https://matrix.org/).

### Documentation

Supported Protocols:

- [Onebot11](https://github.com/botuniverse/onebot-11)

Agents:

[NapCatQQ](https://github.com/NapNeko/NapCatQQ) (tested: v4.4.11)

```json
"messagePostFormat": "array",
"enableLocalFile2Url": true,
```

Some quick links:

- [Bridge setup](https://docs.mau.fi/bridges/go/setup.html)
- [Docker](https://hub.docker.com/r/lxduo/matrix-pylon)

### Features & roadmap

- Matrix → Pylon

  - [ ] Message types
    - [x] Text
    - [x] Image
    - [x] Sticker
    - [x] Video
    - [x] Audio
    - [x] File
    - [x] Mention
    - [x] Reply
    - [x] Location
  - [x] Chat types
    - [x] Direct
    - [x] Room
  - [ ] Presence
  - [x] Redaction
  - [ ] Group actions
    - [ ] Join
    - [ ] Invite
    - [ ] Leave
    - [ ] Kick
    - [ ] Mute
  - [ ] Room metadata
    - [ ] Name
    - [ ] Avatar
    - [ ] Topic
  - [ ] User metadata
    - [ ] Name
    - [ ] Avatar

- Pylon → Matrix

  - [ ] Message types
    - [x] Text
    - [x] Image
    - [x] Sticker
    - [x] Video
    - [x] Audio
    - [x] File
    - [x] Mention
    - [x] Reply
    - [x] Location
  - [ ] Chat types
    - [x] Private
    - [x] Group
  - [ ] Presence
  - [x] Redaction
  - [ ] Group actions
    - [ ] Invite
    - [ ] Join
    - [ ] Leave
    - [ ] Kick
  - [ ] Mute
  - [ ] Group metadata
    - [x] Name
    - [x] Avatar
    - [ ] Topic
  - [x] User metadata
    - [x] Name
    - [x] Avatar
  - [ ] Login types
    - [ ] Password
    - [x] QR code

- Misc
  - [ ] Automatic portal creation
    - [ ] After login
    - [ ] When added to group
    - [x] When receiving message
  - [x] Double puppeting
