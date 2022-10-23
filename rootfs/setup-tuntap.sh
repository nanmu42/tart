set -exo pipefail

# The following settings won't be persistent
# and will be reset after system reboot.

# change this per your local configuration
# e.g. eth0
BACKBONE=wlp6s0

ip tuntap add tap0 mode tap
ip addr add 172.18.0.1/24 dev tap0
ip link set tap0 up
echo 1 > /proc/sys/net/ipv4/ip_forward
iptables -t nat -A POSTROUTING -o "$BACKBONE" -j MASQUERADE
iptables -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
iptables -A FORWARD -i tap0 -o "$BACKBONE" -j ACCEPT