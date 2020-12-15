# Gotracker

## What is it?

It's a tracked music player written in Go.

## Why does this exist?

I needed to learn Go forever ago and figured this was a good way to do it.

## What does it play?

At the moment, just S3M (Screamtracker 3) files and very terribly simulated MOD (Protracker/Fasttracker) files.

## What systems does it work on?

* Windows (Windows 2000 or newer)
  * WinMM (`WAVE_MAPPER` device)
  * File (Wave/RIFF file)
  * PulseAudio (via optional build flag) - NOTE: Not recommended except for WSL (Linux) builds!
* Linux
  * File (Wave/RIFF file)
  * PulseAudio (via optional build flag)

## How do I build this thing?

### What you need

For a Windows build, I recommend the following:
* Windows 2000 (or newer) - I used Windows 10 Pro (Windows 10 Version 20H2)
* Visual Studio Code
  * Go extension for VSCode v0.19.0 (or newer) 
  * Go v1.15.2 (though it will probably compile with Go v1.05 or newer)

For a non-Windows (e.g.: Linux) build, I recommend the following:
* Ubuntu 20.04 (or newer) - I used Ubuntu 20.04.1 LTS running in WSL2
* Go v1.15.2 (or newer)

### How to build (on Windows)

1. First, load the project folder in VSCode.  If this is the first time you've ever opened a Go project, VSCode will splash up a thousand alerts asking to install various things for Go. Allow it to install them before continuing on.
2. Next, open a Terminal for `powershell`.
3. Enter the following command
   ```powershell
   go build
   ```
   When the command completes, you should now have the gotracker.exe file. Drag an .S3M file on top of it!

### How to build (on Linux, without PulseAudio support)

1. Build the player with the following command
   ```bash
   go build
   ```

### How to build (on Linux, with PulseAudio support)

1. Build the player with the following command
   ```bash
   go build -tags=pulseaudio
   ```

## How does it work?

Not well, but it's good enough to play some moderately complex stuff.

## Bugs

### Known bugs

| Tags | Notes |
|--------|---------|
| `s3m` | Unknown/unhandled commands (effects) will cause a panic. There aren't many left, but there are still some laying around. |
| `player` | The rendering system is fairly bad - it originally was designed only to work with S3M, but I decided to rework some of it to be more flexible. I managed to pull most of the mixing functionality out into somewhat generic structures/algorithms, but it still needs a lot of work. |
| `s3m` | Attempting to load a corrupted S3M file may cause the deserializer to panic or go running off into the weeds indefinitely. |
| `mod` | MOD file support is generally terrible. |
| `s3m` | Attempting to play an S3M file with Adlib/OPL2 instruments has unexpected behavior. |
| `windows` `winmm` | Setting the number of channels to more than 2 may cause WinMM and/or Gotracker to do unusual things. You might be able to get a hardware 4-channel capable card (such as the Aureal Vortex 2 AU8830) to work, but driver inconsistencies and weirdnesses in WinMM will undoubtedly cause needless strife. |
| `player` | Channel readouts are associated to the buffer being fed into the output device, so the log line showing the row/channels being played might appear unattached to what's coming from the sound system. |
| `s3m` | Setting the default `C2SPD` value for the `s3m` package to something other than 8363 will cause some unusual behavior - Lower values will reduce the fidelity of the audio, but it will generally sound the same. However, the LFOs (vibrato, tremelo) will become significantly more pronounced the lower the `C2SPD` becomes. The inverse of the observed phenomenon occurs when the `C2SPD` value gets raised. At a certain point much higher than 8363, the LFOs become effectively useless. |
| `player` `mixing` | The mixer still uses some simple saturation mixing techniques, but it's a lot better than it used to be. |
| `pulseaudio` | PulseAudio support is offered through a Pure Go interface originally created by Johann Freymuth, called [jfreymuth/pulse](https://github.com/jfreymuth/pulse). While it seems to work pretty well, it does have some inconsistencies when compared to the FreeDesktop supported C interface. If you see an error about there being a "`missing port in address`", make sure to append the port `:4713` to the end of the `PULSE_SERVER` environment variable. I will create a pull request to their repo soon-ish in hopes to fix this in a reasonable way. |


### Unknown bugs

* There are many, I'm sure.

## Further reading

Take a look at the fmoddoc2 documentation that the folks at FireLight studios released forever ago - it has great info how how to make a mod player, upgrade it to an s3m player, and then dork around with the internals a bit.
