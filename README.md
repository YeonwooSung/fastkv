# FastKV

A distributed fast and lightweight key value store.

## ToDo

- [v] Implement the bloom filter for the key-value store
- [ ] Implement the Zanzibar-like auth checking for security

## Credits

This project is based on [geohot/minikeyvalue](https://github.com/geohot/minikeyvalue).
Credits to [George Hotz](https://github.com/geohot).

For the bloom filter, we refered to [this post](https://itnext.io/bloom-filters-and-go-1d5ac62557de)

As the security and authentication are important for a production-level key-value store, I have referred to [spicedb](https://github.com/authzed/spicedb), which is an open-source implementation of Google Zanzibar.
