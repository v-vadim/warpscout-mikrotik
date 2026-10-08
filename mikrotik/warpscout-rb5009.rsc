# Prepared for mikrrr.rsc: RB5009, RouterOS 7.24.5.
# Upload warpscout-mikrotik-arm64.tar to usb1 before applying.
# Run once; do not import again over an existing warpscout installation.
# Confirm 192.168.254.31 is unused on the live network.

/interface/veth/add name=WARPSCOUT address=192.168.254.31/24 gateway=192.168.254.1 dhcp=no
/interface/bridge/port/add bridge=Bridge-Docker interface=WARPSCOUT

/container/mounts/add list=warpscout src=/usb1/docker_configs/mihomo_mikrotik/awg dst=/output
/container/mounts/add list=warpscout src=/usb1/docker_configs/warpscout/state dst=/state

/container/envs/add list=warpscout key=SCAN_COMMAND value="scan -p awg -tg-only -conf warp_ru_4.conf"
/container/envs/add list=warpscout key=INTERVAL_SECONDS value="21600"

# Existing logging excludes container topics from the general info rule.
/system/logging/add topics=container action=memory prefix="WARPSCOUT-CONTAINERS "

/container/add name=warpscout file=usb1/warpscout-mikrotik-arm64.tar interface=WARPSCOUT dns=1.1.1.1,8.8.8.8 envlists=warpscout mountlists=warpscout root-dir=/usb1/docker/warpscout logging=yes start-on-boot=no workdir=/state

# Wait for extraction to finish, then run manually:
# /container/start [find where name="warpscout"]
# /container/print detail where name="warpscout"
# /log/print where topics~"container"
# /file/print where name~"warp_ru_4.conf"

# No additional NAT, route or LAN list member is needed for the supplied config.
# The exported startup task Remount_USB_Disk resets USB power; test USB readiness
# at reboot before enabling start-on-boot for this container.
