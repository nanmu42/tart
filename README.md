**English** | [中文](https://jihulab.com/nanmu42/tart/-/blob/main/README_zh-cn.md)

# Tart

[![GoDoc](https://godoc.org/github.com/nanmu42/tart?status.svg)](https://godoc.org/github.com/nanmu42/tart)
[![Build status](https://jihulab.com/nanmu42/tart/badges/main/pipeline.svg)](https://jihulab.com/nanmu42/tart/-/pipelines)

<div align="center">
  <img width="360" src="https://user-images.githubusercontent.com/8143068/197567829-b2d9783d-a660-41c6-bea4-5945dfa1ccb3.png">
</div>


Tart is an educational purpose, unofficial Gitlab Runner, implementing a subset of functionality of Gitlab Runner as experiments and demonstration.

For Example, Tart can run [its own CI job, in which its unit tests are executed and its binary got compiled](https://jihulab.com/nanmu42/tart/-/jobs/4980020).

Features:

* Fun!
* Uses [Firecracker](https://firecracker-microvm.github.io/) and `/dev/kvm`. Every job runs in a "microVM" that boots under 2 seconds. Tart might be the first example to combine Gitlab runner and Firecracker
* The codebase is relatively small at around 2000 lines(empty lines included) and the core functionality of Gitlab Runner is implemented: polling jobs, execution in isolation environment, submition of job state and logs

It's a toy runner and functionality like artifact uploading and services are not supported. In other words, don't use it in production.

## Usage

Tart runs in a Linux environment with access to `/dev/kvm`:

```bash
sudo setfacl -m u:${USER}:rw /dev/kvm
```

1. Download binaries of Tart and Firecracker from release page and put them into `$PATH`
2. Download rootFS and Linux kernel from release page and put them into a working directory, such as `~/tart`
3. Create network for microVMs, refer to `rootfs/setup-tuntap.sh`
4. `cd ~/tart`
5. Register Tart as your project CI runner: `tart register --endpoint https://gitlab.example.com --token your_token_here > tart.toml`
6. Run Tart: `tart run`
7. Trigger CI job on Gitlab. You may have to disable shared runner to ensure CI jobs are scheduled to Tart
8. Watch Tart working(or exploding)

## Compile

```bash
make
```

The binary can be found under `bin` directory.

To compile Linux kernel and build rootFS, refer to `rootFS` directory.

## What's with the name?

I like egg tarts.

## License

MIT

Licenses of work of third parties lies at `THIRD_PARTY_LICENSES.md`.

The tart photo in the logo is from Ashley Byrd on Unsplash. Gopher is generated at [gopherize.me].