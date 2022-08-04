<h1 align="center">
    Termbot
</h1>
<p align="center">A fully fledged terminal emulator in a Discord chat, inspired by <a href="https://github.com/Adikso/BashBot">BashBot</a></p>

<p align="center">
    <img src="https://img.shields.io/github/stars/polyzium/termbot.svg?style=for-the-badge&colorB=E8BE5D" alt="Stars" />
    <img src="https://img.shields.io/github/issues/polyzium/termbot?style=for-the-badge" alt="Open issues">
    <img src="https://img.shields.io/badge/Go-1.18+-00ADD8?style=for-the-badge&logo=go" alt="go version" />
    <img src="https://img.shields.io/badge/license-gpl3.0-red?style=for-the-badge&logo=none" alt="license" />
</p>

## ⚡️ Quick start

First, [download](https://golang.org/dl/) and install **Go**. Version `1.18` or later is required.

Now you can clone the repo, and run the project.
```bash
git clone https://github.com/polyzium/termbot
cd termbot
go run .
```
Be sure to [configure](#configuration) your instance before running the bot!

## Features
- Session management
- Interactive mode
- Autosubmit mode
- Execution of a single command
- Macros
- TUI friendly
- Color!  

**Color is experimental!** As it uses ANSI escape codes the bot can easily reach Discord's 2000 character limit.

## Usage
User settings are managed via slash commands. Terminal input is being done via the prefix (unless interactive mode is enabled).  
The bot accepts shortcut sequences for keys. See the list below.

## Shortcuts
Input | Key name
--- | ---
\n | Linefeed
\r | Carriage Return (Enter)
\b | Backspace
\t | Tab
[ESC] | Escape
[F1] | F1
[F2] | F2
[F3] | F3
[F4] | F4
[F5] | F5
[F6] | F6
[F7] | F7
[F8] | F8
[F9] | F9
[F10] | F10
[F11] | F11
[F12] | F12
[UP] | Up Arrow
[DOWN] | Down Arrow
[RIGHT] | Right Arrow
[LEFT] | Left Arrow
[INS] | Insert
[DEL] | Delete
[PGUP] | Page Up
[PGDN] | Page Down
^[key] | CTRL + key (i.e. ^C = CTRL+C, etc)

## Configuration
The configuration values are located in the config.yaml file. The keys should be pretty self explanatory.  

Example:
```yaml
token: YOUR_TOKEN_HERE
prefix: $
ownerid: "YOUR_USERID_HERE"
macros:
    - name: example
      in: Here's an example macro.
      whitelist: false
      allowedids: []
userprefs:
    "552930095141224479":
        defaultsharedusers:
            - "216836179415269376"
        color: false
        interactive: true
        autosubmit: true
```

## Made with <3 using
[DiscordGo](https://github.com/bwmarrin/discordgo)  
[pty for Go](https://github.com/creack/pty)  
[vt10x](https://github.com/hinshun/vt10x)

## Thanks to,
[anic17](https://github.com/anic17) for testing  
[ZackeryRSmith](https://github.com/ZackeryRSmith) for suggestions, and README

<p align="right">
<sub>(<b>Termbot</b> is protected by the <a href="https://github.com/polyzium/termbot/blob/master/LICENSE"><i>GPLv3</i></a> licence)</sub>
</p>
