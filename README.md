# SuperfastKV

A persistent, embeddable key-value storage engine written in Go. Built from scratch using first principles.

This project was built for the sake of learning how storage engines work under the hood. <br>
And so correctness > speed (for V1).

## What it is

A storage engine built to understand what happens underneath databases. No frameworks, no abstractions borrowed from elsewhere. Every component that you see in this repo are implemented from scratch. The name is aspirational I know, but v1 is only about getting it right.

## Project Layout

Pretty messed up for now and it remains this way until we get to setting up the KV store.

## V1

- [ ] Persistent CoW B-tree: CoW enabling lock-free reads
- [ ] Raw syscall (mmap + pwrite) I/O: no buffered stdlib, mmap for reads, pwrite for atomic page writes.
- [ ] CRC32 corruption detection: every page is checksummed on write and verified on read.
- [ ] Free list management: reclaims pages from deleted or CoW-replaced nodes.
- [ ] Write-ahead log (WAL): crash recovery via a sequential log.
- [ ] fsync durability: ensure data survives OS crashes, not just process crashes.
- [ ] Benchmarks: and maybe comparing it with other KV stores

## V2

- [ ] SIMD-accelerated comparisons: vectorized key search within nodes, better cache utilization.
- [ ] TOAST-style storage: `The Oversized-Attribute Storage Technique (TOAST)` by Postgres
- [ ] Stratified B-tree: separate hot and cold layers, 100x faster writes, 10x faster reads
- [ ] More to be determined: I don't know what I don't know
