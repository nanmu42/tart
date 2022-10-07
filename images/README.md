# Firecracker RootFS and Kernel

## Kernel

To build [a Kernel supported by Firecracker](https://github.com/firecracker-microvm/firecracker/blob/main/docs/kernel-policy.md), follow the [guide](https://github.com/firecracker-microvm/firecracker/blob/main/docs/rootfs-and-kernel-setup.md#manual-compilation) and [configurations](https://github.com/firecracker-microvm/firecracker/blob/main/resources/guest_configs).

Use `make vmlinux -j$(nproc)` can accelerate the building process.

## RootFS

```bash
bash build-jammy.sh
```

## References

* https://github.com/firecracker-microvm/firecracker/blob/main/docs/rootfs-and-kernel-setup.md
* https://jvns.ca/blog/2021/01/23/firecracker--start-a-vm-in-less-than-a-second/