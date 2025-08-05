# RTX 4080 Bitcoin & Altcoin CUDA GPU Miner

<div align="center">
  <img src="https://raw.githubusercontent.com/inandoutofthebox/Github-Profiles-Script/refs/heads/main/Logo.jpg" width="250"/>
  <br><br>
  <b>Ultra-optimized CUDA Miner for Bitcoin and Many Altcoins</b><br>
  <i>Fully designed for NVIDIA RTX 4080 GPU · Direct CUDA & Go Integration · Windows 10/11 & Linux</i>
  <i>Does this makes senseat all? No not at all theres better Hardware for more efficient Cryptomining results. This is just a POC since GPU mining on newer NVIDIA Hardware wasnt possible...</i>
</div>

---

## ✨ Features

- **Direct GPU Mining** – CUDA kernels for SHA-256 (Bitcoin), Scrypt, Ethash, Equihash, RandomX, X11, KAWPOW, BLAKE2b, Lyra2REv3, Cuckatoo32, and more.
- **Optimized for RTX 4080** – Utilizes all 9,728 CUDA Cores, SM architecture, shared memory, and registers.
- **Go-Binding** – High-level mining control and benchmarking via Go frontend (`miner.go`) with live stats.
- **Fast, Configurable Nonce Range** – Extremely fast, user-configurable threads per mining round (default 256K up to >1 million).
- **Cross-Platform Implementation** – Validated for Windows 10/11 with CUDA 12.x and native Linux.
- **Detailed Debug Output, Performance Stats, Safe Shutdown.**

---

## 🔋 Requirements

- **NVIDIA RTX 4080** (or compatible card for maximum performance)
- **CUDA Toolkit 12.x**
- **Go (>=1.18)**
- **Windows 10/11** or **Ubuntu 22.04+**
- gcc/g++ and go installed

---

## 📁 Project Structure

- **bitcoin_cuda.cu**  
  Complete CUDA implementation of all mining algorithms, Windows/Linux-compatible CUDA exports, C-API for Go binding.

- **miner.go**  
  Go frontend for mining control, block header creation, performance monitor, CLI menu, benchmark & stats. Calls CUDA kernels via Cgo.

---

## 🚀 Compilation

### 1. Build the CUDA library (`bitcoin_cuda.cu`)

```

nvcc -O3 -arch=sm_89 -Xcompiler -fPIC -shared -o libbitcoin_cuda.so bitcoin_cuda.cu

# On Windows:

nvcc -O3 -Xcompiler /LD -o bitcoin_cuda.dll bitcoin_cuda.cu

```

### 2. Build the Go program (`miner.go`)

```

go build -o rtx4080_miner miner.go

```

(*The Go code expects `libbitcoin_cuda.so` to be in the current directory or system library path.*)

---

## ⚡ Usage

Run the mining program:

```

./rtx4080_miner

```

You will get a menu with these modes:

1. **GPU Solo Mining Test**  
   Mines a single (test) block at low difficulty.

2. **GPU Benchmark (30/60 seconds)**  
   Measures and shows the hash rate under full load, shows live results and efficiency.

3. **Continuous Mining**  
   Continuous mining with evaluation; ideal for long-run or pool mining customization.

4. **Debug Mode**  
   Shows all internal parameters, nonce ranges, and live debug info each round.

5. **Custom Difficulty**  
   Create and mine a block with a self-selected difficulty from a list.

Mining can be safely stopped at any time using `CTRL+C`.

---

## 🛠️ Core Features (Overview)

- **CUDA kernels for all algorithms** (SHA-256, Scrypt, Ethash, X11, …), optimized for RTX 4080: block config, warp/SM params, shared memory tricks.
- **C-API (`EXPORT` functions)** for easy integration (also callable in other programming languages).
- **Go statistics module:** Live hashrate, blocks found, time, best hash, power efficiency stats.
- **User-friendly, colored console output:** In the Go miner including progress, live statistics, and debug info.

---

## 📊 Output & Metrics

- **Hashrate:** Displayed live in MH/s.
- **Blocks found:** With solution details (nonce, hash, time taken).
- **Best hash:** Lowest-ever calculated hash for fine-tuning/statistics.
- **Energy efficiency:** Output in MH/W (default based on RTX 4080: 320W).

---

## 🔎 Example Output: Startup & Benchmarks

```

🚀 Bitcoin GPU-Miner for RTX 4080
==================================
💻 System Information:
OS: linux
Arch: amd64
CPUs: 16
Go Version: go1.21.0
GPU: RTX 4080 (assumed)

Mining Modes:

1. 🔧 GPU Solo Mining (Standard Test)
2. ⚡ GPU Benchmark (30 seconds)
3. 🎯 GPU Benchmark (60 seconds)
4. 🔄 Continuous Mining
5. 🐛 Debug Mode (with detailed output)
6. 🎛️ Custom Difficulty

Choose mode (1-6):

```

When a block is found, a full summary including nonce, hash, time, rate, and efficiency is shown.

---

## 🧑‍💻 API / Extensions

Thanks to the C exports in the CUDA code, you can call the mining kernels directly from other languages (Python, C#, Rust, etc.) – just load the compiled library dynamically.

---

## 💡 Tips & Notes

- Difficulty (bits) is set to "very easy" for demo purposes – please adjust for real-world mining.
- The RTX 4080 can achieve enormous hashrates (~500 MH/s+ in SHA-256). Actual numbers depend on OC/TDP/drivers.
- Mining may require root/administrator rights – especially on Windows to access the driver.
- For multi-algo: choose the CUDA algorithm in the start menu as needed.

---

## 🧩 Roadmap / Planned Features

- Native pool mining (Stratum) support
- Dynamic difficulty adjustment
- Power measurement via NVML
- Improved output formats (JSON, Prometheus)
- ARM/MacBook M1 CUDA-backend (experimental)
- Dockerfile and CI/CD builds

---

## 📝 License

MIT License – Open Source. Optimized and documented by me [@inandoutofthebox](https://github.com/inandoutofthebox)  
Feedback, bug reports, and pull requests always welcome!

---

## 👋 Contact & Support

- Please submit issues directly on the GitHub repo
- For feature requests: open an issue or pull request!

---

**TL;DR:**  
This repo contains a fully standalone, cross-platform, GPU hashminer (Bitcoin/Altcoins, RTX 4080+), provided as a CUDA library and Go CLI. Run mining tests, benchmarks and development without relying on heavyweight mining suites!

See the source files `bitcoin_cuda.cu` and `miner.go` for all technical details.  
Questions? Just open an issue in the repo!

