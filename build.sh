#!/usr/bin/env bash

case $1 in
  run-gui)
    if [[ $2 == "theme-light" ]]; then
      GTK_THEME="Adwaita:light"
    else
      GTK_THEME="Adwaita:dark"
    fi
    go build -mod=vendor -v -o dist/clyp_linux_amd64_v1/clyp . \
      && go build -mod=vendor -v -o dist/clyp_linux_amd64_v1/clyp-watcher ./cmd/clyp-watcher \
      && cd dist/clyp_linux_amd64_v1/ \
      && GTK_THEME=$GTK_THEME RUN_ENV=dev ./clyp
    ;;
  run-watcher)
    go build -mod=vendor -v -o dist/clyp_linux_amd64_v1/clyp-watcher ./cmd/clyp-watcher \
      && cd dist/clyp_linux_amd64_v1/ \
      && ./clyp-watcher
    ;;
  package-snapshot)
    goreleaser release --snapshot --clean \
      && nfpm pkg --packager deb --target dist/ \
      && nfpm pkg --packager archlinux --target dist/ \
      && nfpm pkg --packager rpm --target dist/
    ;;
  release)
    # $2 old version
    # $3 new version
    # #4 environment variable
    sed -i "s/$2/$3/g" README.md \
      && sed -i "s/$2/$3/g" app.go \
      && sed -i "s/$2/$3/g" nfpm.yaml \
      && git add . \
      && git commit -m "Bump version to $3" \
      && git tag $3 \
      && git push \
      && git push --tags \
      && goreleaser build --clean \
      && nfpm pkg --packager deb --target dist/ \
      && nfpm pkg --packager archlinux --target dist/ \
      && nfpm pkg --packager rpm --target dist/ \
      && GITHUB_TOKEN=$4 goreleaser release --clean
    ;;
esac
