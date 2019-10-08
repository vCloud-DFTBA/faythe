# Faythe cluster

## Idea

Instances of a Faythe cluster communicate on top of a gossip protocol managed via [Hashicorp Memberlist](https://github.com/hashicorp/memberlist).

Faythe instances use the gossip layer to:
* Keep track of membership.
* Combining with [Consistent hashing](https://en.wikipedia.org/wiki/Consistent_hashing) - [Chord](https://en.wikipedia.org/wiki/Chord_(peer-to-peer)), execute scalers.

