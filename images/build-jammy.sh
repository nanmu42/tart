sudo umount /tmp/my-rootfs || true
mkdir -p /tmp/my-rootfs
truncate -s 1G jammy.rootfs.ext4
mkfs.ext4 -F jammy.rootfs.ext4
sudo mount jammy.rootfs.ext4 /tmp/my-rootfs

docker rm -f jammy-rootfs || true
docker run -i --name jammy-rootfs -h tart-jammy ubuntu:jammy bash -s < ./in-container-setup.sh

dirs="bin etc home lib lib64 opt root sbin usr"
for d in $dirs; do sudo docker cp jammy-rootfs:/"$d" /tmp/my-rootfs; done

sudo umount /tmp/my-rootfs
docker rm -f jammy-rootfs
