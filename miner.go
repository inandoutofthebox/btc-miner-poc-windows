package main

/*
#cgo LDFLAGS: -L. -lbitcoin_cuda -lcudart
#include <stdint.h>
extern int cuda_mine_wrapper(uint8_t *header, uint32_t nonce_start, uint32_t *target, 
                             uint32_t *result, uint32_t threads);
*/

#define RTX4080_SM_COUNT        76
#define RTX4080_CORES_PER_SM    128  
#define BLOCK_SIZE              256
#define GRID_SIZE               (RTX4080_SM_COUNT * 2)


import "C"

import (
    "bytes"
    "crypto/sha256"
    "encoding/binary"
    "fmt"
    "math/big"
    "os"
    "os/signal"
    "runtime"
    "sync"
    "syscall"
    "time"
    "unsafe"
)

// ==================== BLOCK HEADER ====================

type BlockHeader struct {
    Version        uint32
    PrevBlockHash  [32]byte
    MerkleRoot     [32]byte
    Timestamp      uint32
    Bits           uint32
    Nonce          uint32
}

func (bh *BlockHeader) Serialize() []byte {
    var buffer bytes.Buffer
    
    binary.Write(&buffer, binary.LittleEndian, bh.Version)
    buffer.Write(bh.PrevBlockHash[:])
    buffer.Write(bh.MerkleRoot[:])
    binary.Write(&buffer, binary.LittleEndian, bh.Timestamp)
    binary.Write(&buffer, binary.LittleEndian, bh.Bits)
    binary.Write(&buffer, binary.LittleEndian, bh.Nonce)
    
    return buffer.Bytes()
}

func (bh *BlockHeader) Hash() [32]byte {
    serialized := bh.Serialize()
    hash1 := sha256.Sum256(serialized)
    hash2 := sha256.Sum256(hash1[:])
    return hash2
}

func (bh *BlockHeader) String() string {
    return fmt.Sprintf("BlockHeader{Version:%d, Timestamp:%d, Bits:%08x, Nonce:%d}", 
        bh.Version, bh.Timestamp, bh.Bits, bh.Nonce)
}

// ==================== MINING STATISTICS ====================

type MiningStats struct {
    HashesTried     uint64
    BlocksFound     uint64
    StartTime       time.Time
    LastUpdate      time.Time
    BestHash        uint32
    ClosestAttempts []uint32
    TotalPower      float64
    mutex           sync.RWMutex
}

func (ms *MiningStats) IncrementHashes(count uint64) {
    ms.mutex.Lock()
    defer ms.mutex.Unlock()
    ms.HashesTried += count
    ms.LastUpdate = time.Now()
}

func (ms *MiningStats) GetHashes() uint64 {
    ms.mutex.RLock()
    defer ms.mutex.RUnlock()
    return ms.HashesTried
}

func (ms *MiningStats) IncrementBlocks() {
    ms.mutex.Lock()
    defer ms.mutex.Unlock()
    ms.BlocksFound++
}

func (ms *MiningStats) UpdateBestHash(hash uint32) {
    ms.mutex.Lock()
    defer ms.mutex.Unlock()
    if hash < ms.BestHash || ms.BestHash == 0 {
        ms.BestHash = hash
        ms.ClosestAttempts = append(ms.ClosestAttempts, hash)
        if len(ms.ClosestAttempts) > 10 {
            ms.ClosestAttempts = ms.ClosestAttempts[1:]
        }
    }
}

func (ms *MiningStats) GetDetailedStats() (uint64, uint64, time.Duration, float64, uint32) {
    ms.mutex.RLock()
    defer ms.mutex.RUnlock()
    
    elapsed := time.Since(ms.StartTime)
    hashRate := float64(ms.HashesTried) / elapsed.Seconds()
    
    return ms.HashesTried, ms.BlocksFound, elapsed, hashRate, ms.BestHash
}

// ==================== GPU MINER ====================

type GPUMiner struct {
    Target        *big.Int
    Stats         *MiningStats
    Running       bool
    Debug         bool
    ThreadsPerRun uint32
    mutex         sync.RWMutex
}

func NewGPUMiner(debug bool) *GPUMiner {
    return &GPUMiner{
        Stats:         &MiningStats{},
        Debug:         debug,
        ThreadsPerRun: 1024 * 256, // 256K Threads pro Iteration
    }
}

func (gm *GPUMiner) GPUMine(header *BlockHeader) (*BlockHeader, error) {
    if gm.Debug {
        fmt.Printf("🔍 DEBUG: Mining-Details\n")
        fmt.Printf("📋 %s\n", header.String())
        fmt.Printf("🎯 Target: %064x\n", gm.Target)
        fmt.Printf("⚡ Threads pro Run: %d\n", gm.ThreadsPerRun)
    }
    
    fmt.Printf("🚀 Starte GPU Mining mit RTX 4080\n")
    
    gm.SetRunning(true)
    gm.Stats.StartTime = time.Now()
    
    // Graceful Shutdown Handler
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-c
        fmt.Printf("\n⚠️  Shutdown-Signal empfangen. Beende Mining...\n")
        gm.SetRunning(false)
    }()
    
    // Statistik-Output
    go gm.printStats()
    
    nonce := uint32(0)
    roundCount := 0
    
    for gm.IsRunning() {
        roundCount++
        if gm.Debug {
            fmt.Printf("🔄 Mining-Runde %d, Nonce-Start: %d\n", roundCount, nonce)
        }
        
        headerBytes := header.Serialize()
        
        // Target für CUDA (vereinfacht)
        var target uint32 = 0x0000FFFF
        var result uint32
        
        // CUDA Mining über externe Library aufrufen
        cudaResult := C.cuda_mine_wrapper(
            (*C.uint8_t)(unsafe.Pointer(&headerBytes[0])),
            C.uint32_t(nonce),
            (*C.uint32_t)(unsafe.Pointer(&target)),
            (*C.uint32_t)(unsafe.Pointer(&result)),
            C.uint32_t(gm.ThreadsPerRun),
        )
        
        if cudaResult != 0 {
            return nil, fmt.Errorf("CUDA Mining Fehler: %d", cudaResult)
        }
        
        gm.Stats.IncrementHashes(uint64(gm.ThreadsPerRun))
        
        // Lösung gefunden?
        if result != 0 {
            header.Nonce = result
            gm.Stats.IncrementBlocks()
            
            hashes, blocks, elapsed, hashRate, bestHash := gm.Stats.GetDetailedStats()
            
            fmt.Printf("\n✅ BLOCK GEFUNDEN mit RTX 4080! 🎉\n")
            fmt.Printf("═══════════════════════════════════════\n")
            fmt.Printf("🔑 Gewinnende Nonce: %d\n", header.Nonce)
            fmt.Printf("📊 Finaler Hash: %064x\n", header.Hash())
            fmt.Printf("🎯 Target erreicht: %x\n", target)
            fmt.Printf("💪 Gesamte Versuche: %d\n", hashes)
            fmt.Printf("⚡ Hash Rate: %.2f MH/s\n", hashRate/1000000)
            fmt.Printf("⏱️  Gesamt-Zeit: %v\n", elapsed)
            fmt.Printf("🏆 Gefundene Blöcke: %d\n", blocks)
            fmt.Printf("📈 Bester Hash: %x\n", bestHash)
            fmt.Printf("═══════════════════════════════════════\n")
            
            return header, nil
        }
        
        nonce += gm.ThreadsPerRun
        
        // Nonce Overflow prüfen
        if nonce < gm.ThreadsPerRun {
            if gm.Debug {
                fmt.Printf("🔄 Nonce Overflow - Timestamp Update\n")
            }
            header.Timestamp = uint32(time.Now().Unix())
            nonce = 0
        }
    }
    
    return nil, fmt.Errorf("GPU Mining gestoppt")
}

func (gm *GPUMiner) SetRunning(running bool) {
    gm.mutex.Lock()
    defer gm.mutex.Unlock()
    gm.Running = running
}

func (gm *GPUMiner) IsRunning() bool {
    gm.mutex.RLock()
    defer gm.mutex.RUnlock()
    return gm.Running
}

func (gm *GPUMiner) printStats() {
    ticker := time.NewTicker(3 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if !gm.IsRunning() {
                return
            }
            
            hashes, blocks, elapsed, hashRate, bestHash := gm.Stats.GetDetailedStats()
            
            if gm.Debug {
                fmt.Printf("🔥 RTX 4080: %.1f MH/s, Hashes: %d, Zeit: %v\n", 
                    hashRate/1000000, hashes, elapsed)
                fmt.Printf("   💾 Threads: %d, Blöcke: %d, Bester Hash: %x\n", 
                    gm.ThreadsPerRun, blocks, bestHash)
                fmt.Printf("   📊 Effizienz: %.2f MH/W (bei ~320W)\n", hashRate/1000000/320)
            } else {
                fmt.Printf("🔥 RTX 4080: %.1f MH/s, Hashes: %d, Zeit: %v\n", 
                    hashRate/1000000, hashes, elapsed)
            }
        }
    }
}

// ==================== BENCHMARK FUNKTIONEN ====================

func (gm *GPUMiner) RunBenchmark(duration time.Duration) {
    fmt.Printf("⚡ Starte GPU Benchmark für %v\n", duration)
    
    header := CreateTestBlockHeader()
    gm.Target = BitsToTarget(header.Bits)
    
    // Benchmark-Timer
    go func() {
        time.Sleep(duration)
        gm.SetRunning(false)
    }()
    
    startTime := time.Now()
    result, err := gm.GPUMine(header)
    actualDuration := time.Since(startTime)
    
    hashes, blocks, _, hashRate, bestHash := gm.Stats.GetDetailedStats()
    
    fmt.Printf("\n📊 RTX 4080 Benchmark Ergebnisse:\n")
    fmt.Printf("═══════════════════════════════════════\n")
    fmt.Printf("⏱️  Laufzeit: %v\n", actualDuration)
    fmt.Printf("🔢 Gesamte Hashes: %d\n", hashes)
    fmt.Printf("⚡ Durchschnittliche Hash Rate: %.2f MH/s\n", hashRate/1000000)
    fmt.Printf("🏆 Gefundene Blöcke: %d\n", blocks)
    fmt.Printf("📈 Bester Hash: %x\n", bestHash)
    fmt.Printf("💡 Theoretische BTC/Tag: %.12f\n", hashRate*86400/1e18)
    fmt.Printf("⚡ Stromeffizienz: %.2f MH/W\n", hashRate/1000000/320)
    fmt.Printf("═══════════════════════════════════════\n")
    
    if err != nil && err.Error() != "GPU Mining gestoppt" {
        fmt.Printf("❌ Benchmark Fehler: %v\n", err)
    }
    
    if result != nil {
        fmt.Printf("🎉 Block während Benchmark gefunden!\n")
    }
}

// ==================== UTILITY FUNKTIONEN ====================

func BitsToTarget(bits uint32) *big.Int {
    exponent := uint8(bits >> 24)
    mantissa := bits & 0x00ffffff
    
    if exponent <= 3 {
        mantissa >>= (8 * (3 - exponent))
        return big.NewInt(int64(mantissa))
    }
    
    target := big.NewInt(int64(mantissa))
    target.Lsh(target, uint(8*(exponent-3)))
    return target
}

func CreateTestBlockHeader() *BlockHeader {
    return &BlockHeader{
        Version:       1,
        PrevBlockHash: [32]byte{},
        MerkleRoot:    sha256.Sum256([]byte("rtx4080_gpu_mining_test")),
        Timestamp:     uint32(time.Now().Unix()),
        Bits:          0x1d00ffff, // Niedrige Schwierigkeit für Tests
        Nonce:         0,
    }
}

func CreateCustomBlockHeader(difficulty uint32) *BlockHeader {
    return &BlockHeader{
        Version:       1,
        PrevBlockHash: [32]byte{},
        MerkleRoot:    sha256.Sum256([]byte("custom_difficulty_test")),
        Timestamp:     uint32(time.Now().Unix()),
        Bits:          difficulty,
        Nonce:         0,
    }
}

func printSystemInfo() {
    fmt.Printf("💻 System-Informationen:\n")
    fmt.Printf("   OS: %s\n", runtime.GOOS)
    fmt.Printf("   Arch: %s\n", runtime.GOARCH)
    fmt.Printf("   CPUs: %d\n", runtime.NumCPU())
    fmt.Printf("   Go Version: %s\n", runtime.Version())
    fmt.Printf("   GPU: RTX 4080 (angenommen)\n")
}

// ==================== MAIN FUNKTION ====================

func main() {
    fmt.Println("🚀 Bitcoin GPU-Miner für RTX 4080")
    fmt.Println("==================================")
    
    printSystemInfo()
    
    fmt.Println("\nMining-Modi:")
    fmt.Println("1. 🔧 GPU Solo Mining (Standard Test)")
    fmt.Println("2. ⚡ GPU Benchmark (30 Sekunden)")
    fmt.Println("3. 🎯 GPU Benchmark (60 Sekunden)")
    fmt.Println("4. 🔄 Kontinuierliches Mining")
    fmt.Println("5. 🐛 Debug-Modus (mit detaillierten Ausgaben)")
    fmt.Println("6. 🎛️  Angepasste Schwierigkeit")
    
    var choice int
    fmt.Print("\nWähle Modus (1-6): ")
    fmt.Scanf("%d", &choice)
    
    switch choice {
    case 1:
        fmt.Println("\n🔧 Starte GPU Solo Mining Test...")
        miner := NewGPUMiner(false)
        header := CreateTestBlockHeader()
        miner.Target = BitsToTarget(header.Bits)
        
        result, err := miner.GPUMine(header)
        if err != nil {
            fmt.Printf("❌ Fehler: %v\n", err)
            return
        }
        
        if result != nil {
            fmt.Printf("🎉 GPU Mining erfolgreich abgeschlossen!\n")
        }
        
    case 2:
        fmt.Println("\n⚡ Starte GPU Benchmark (30 Sekunden)...")
        miner := NewGPUMiner(false)
        miner.RunBenchmark(30 * time.Second)
        
    case 3:
        fmt.Println("\n⚡ Starte GPU Benchmark (60 Sekunden)...")
        miner := NewGPUMiner(false)
        miner.RunBenchmark(60 * time.Second)
        
    case 4:
        fmt.Println("\n🔄 Starte kontinuierliches Mining...")
        fmt.Println("Drücke Ctrl+C zum Stoppen")
        
        miner := NewGPUMiner(false)
        header := CreateTestBlockHeader()
        miner.Target = BitsToTarget(header.Bits)
        
        result, err := miner.GPUMine(header)
        if err != nil {
            fmt.Printf("❌ Fehler: %v\n", err)
            return
        }
        
        if result != nil {
            fmt.Printf("🎉 Kontinuierliches Mining erfolgreich!\n")
        }
        
    case 5:
        fmt.Println("\n🐛 Starte Debug-Modus...")
        miner := NewGPUMiner(true)
        header := CreateTestBlockHeader()
        miner.Target = BitsToTarget(header.Bits)
        
        result, err := miner.GPUMine(header)
        if err != nil {
            fmt.Printf("❌ Fehler: %v\n", err)
            return
        }
        
        if result != nil {
            fmt.Printf("🎉 Debug-Mining erfolgreich!\n")
        }
        
    case 6:
        fmt.Println("\n🎛️  Angepasste Schwierigkeit...")
        
        fmt.Println("Verfügbare Schwierigkeitsgrade:")
        fmt.Println("1. Sehr einfach (0x1d00ffff)")
        fmt.Println("2. Einfach (0x1d0fffff)")
        fmt.Println("3. Mittel (0x1d000fff)")
        fmt.Println("4. Schwer (0x1d0000ff)")
        fmt.Println("5. Sehr schwer (0x1d00000f)")
        
        var diffChoice int
        fmt.Print("Wähle Schwierigkeit (1-5): ")
        fmt.Scanf("%d", &diffChoice)
        
        var difficulty uint32
        switch diffChoice {
        case 1:
            difficulty = 0x1d00ffff
        case 2:
            difficulty = 0x1d0fffff
        case 3:
            difficulty = 0x1d000fff
        case 4:
            difficulty = 0x1d0000ff
        case 5:
            difficulty = 0x1d00000f
        default:
            difficulty = 0x1d00ffff
        }
        
        miner := NewGPUMiner(true)
        header := CreateCustomBlockHeader(difficulty)
        miner.Target = BitsToTarget(header.Bits)
        
        result, err := miner.GPUMine(header)
        if err != nil {
            fmt.Printf("❌ Fehler: %v\n", err)
            return
        }
        
        if result != nil {
            fmt.Printf("🎉 Mining mit angepasster Schwierigkeit erfolgreich!\n")
        }
        
    default:
        fmt.Println("❌ Ungültige Auswahl")
        return
    }
    
    fmt.Println("\n👋 GPU Mining beendet")
    fmt.Println("Danke für die Nutzung des RTX 4080 Bitcoin-Miners!")
}

