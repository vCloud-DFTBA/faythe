# etcd-adapter

[![Build Status](https://travis-ci.org/ntk148v/etcd-adapter.svg?branch=master)](https://travis-ci.org/ntk148v/etcd-adapter)
[![Coverage Status](https://coveralls.io/repos/github/ntk148v/etcd-adapter/badge.svg)](https://coveralls.io/github/ntk148v/etcd-adapter)
[![Godoc](https://godoc.org/github.com/ntk148v/etcd-adapter?status.svg)](https://godoc.org/github.com/ntk148v/etcd-adapter)

ETCD adapter is the policy storage adapter for [Casbin](https://github.com/casbin/casbin). With this library, Casbin can load policy from ETCD and save policy to it. ETCD adapter support the __Auto-Save__ feature for Casbin policy. This means it can support:
- Add a single policy rule to the storage
- Remove a single policy rule from the storage.
- Add a set of policies rule to the storage.
- Remove a set of policies rule to the storage.

Additional, this adapter allows user to:
- Fully control the Etcd configuration. It will be useful if Etcd is enable authentication.
- Support Etcd namespace.

## Installation
```bash
go get github.com/ntk148v/etcd-adapter
```

## Sample Example

Check the [examples folder](./examples)

## Getting Help

- [Casbin](https://github.com/casbin/casbin)

## License

This project is under Apache 2.0 License. See the [LICENSE](LICENSE) file for the full license text.
