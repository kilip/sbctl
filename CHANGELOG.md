# Changelog

## 1.0.0 (2026-05-25)


### Features

* **config:** add vault identity and refactor gitsync ([f8125f7](https://github.com/kilip/sbctl/commit/f8125f729c030acc606ed9deba6d16b9d35a5651))
* **config:** implement configuration management and logging ([4bba47d](https://github.com/kilip/sbctl/commit/4bba47d3b06ee2afea6f5c869915c57085a77363))
* **config:** use testdata/default as dev environment ([bb49ee9](https://github.com/kilip/sbctl/commit/bb49ee91be462f713750e7b88c5edd916d6cbe13))
* **daemon:** complete service implementation and config updates ([7b962e9](https://github.com/kilip/sbctl/commit/7b962e9b6bab98381ffd04ab664c4fb1cdfabeae))
* **daemon:** implement cross-platform background service ([4026f6f](https://github.com/kilip/sbctl/commit/4026f6f0b2dab54795b6dee44d4e069dd48967bf))
* **daemon:** improve logging and update agent standards ([345074f](https://github.com/kilip/sbctl/commit/345074ff3aff251cfad12c2b075d14ee7eab626e))
* **doctor:** implement health check command ([aa66d12](https://github.com/kilip/sbctl/commit/aa66d12538c5d1d90e0a7e71dcd2e1c0b98e00eb))
* **gitsync:** add ssh subcommand for zero-dependency key generation ([852bbe9](https://github.com/kilip/sbctl/commit/852bbe91aab6304fed3140287448e8fd1ccb5c81))
* **gitsync:** automate ssh signing and update project standards ([0a4bf14](https://github.com/kilip/sbctl/commit/0a4bf144a5626fec43b26a5c0e17da4e436a5598))
* **gitsync:** implement automatic vault synchronization with go-git ([47a4008](https://github.com/kilip/sbctl/commit/47a400839bf12ed3970deae07a4ad46a8d607920))
* **setup:** implement interactive wizard and service management ([3456ce6](https://github.com/kilip/sbctl/commit/3456ce64c5f2fb3a957e712ecfa52584f485d1ce))


### Bug Fixes

* **daemon:** resolve data race in reloadWorkers and tests ([0b425b1](https://github.com/kilip/sbctl/commit/0b425b181b93c8b84f66486ea87cdd12c1d3388b))
