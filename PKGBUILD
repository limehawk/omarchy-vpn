# Maintainer: Limehawk <128890849+limehawk@users.noreply.github.com>
pkgname=omarchy-vpn
pkgver=0.4.6
pkgrel=1
pkgdesc="WireGuard VPN manager TUI for Omarchy"
arch=('x86_64')
url="https://github.com/limehawk/omarchy-vpn"
license=('MIT')
depends=('wireguard-tools' 'systemd-resolvconf')
optdepends=('netbird: NetBird mesh VPN row'
            'cloudflare-warp-bin: Cloudflare WARP row')
makedepends=('go')
install=omarchy-vpn.install

build() {
    cd "$startdir"
    go build -ldflags "-X main.version=$pkgver" -o "$srcdir/omarchy-vpn" .
}

package() {
    cd "$srcdir"

    # Binary
    install -Dm755 omarchy-vpn "$pkgdir/usr/bin/omarchy-vpn"

    # Root helper: the only command sudo allows. It takes config names,
    # never paths, and refuses imports that carry root hook commands.
    install -Dm755 "$startdir/omarchy-vpn-helper" "$pkgdir/usr/lib/omarchy-vpn/helper"

    # Sudoers for passwordless WireGuard management (dir mode matches the sudo package)
    install -dm750 "$pkgdir/etc/sudoers.d"
    install -Dm440 /dev/stdin "$pkgdir/etc/sudoers.d/omarchy-vpn" << 'EOF'
%wheel ALL=(root) NOPASSWD: /usr/lib/omarchy-vpn/helper
EOF
}
