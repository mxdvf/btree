Calling it superfastkv would mean nothing if don't measure speed, latency, throughput and write speed and all that.
We'll also measure the actual speed and benchmark it with other in-memory stores like Redis, Memcache, especially Pogocache and also DiceDB.

Tips on how to make the implementation faster:
- Replace linear scan with: binary search (large node fanout) or SIMD scan (if you go hardcore)
- Keep node size aligned to cache lines (typically 64B / 128B)
- Store keys in a contiguous array (you likely already do)
- Consider branchless comparisons

It's going to be fun. Every step of the journey could be an article:
- Building production-grade BTree implementation
- Benchmarking and testing your BTree implementation (because it's literally what powers the entire storage system)
- Benchmarking your kv store, comparing with other stores (can say you built something slower than all others but obviously i am still proud of it)
