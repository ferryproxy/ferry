# Ferry

Ferry is a Kubernetes multi-cluster communication component that eliminates 
communication differences between clusters as if they were in a single cluster,
regardless of the network environment those clusters are in.

## Why Ferry

- Avoid Cloud Lock-in
    - Open up inter-access between different clouds
    - Migration of Service to different clouds is seamless
- Out of the Box
    - Command line tools are provided for easy installation and use
    - Centrally defined rules
- No Intrusion
    - No dependency on Kubernetes version
    - No dependency on any CNI or network environment
    - No need to modify existing environment
- Intranet Traversal
    - Only one public IP is required

## Quick Start

Download  the [ferryctl](https://github.com/ferryproxy/ferry/releases) binary.

Read the [ferryproxy.io](https://ferryproxy.io) for more about Ferry.

## License

Licensed under the Apache 2.0 License. See [LICENSE](https://github.com/ferryproxy/ferry/blob/master/LICENSE) for the full license text.
