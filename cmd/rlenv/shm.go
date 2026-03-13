package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"syscall"
	"unsafe"
)

const (
	shmMagic      = "HSHM"
	shmVersion    = 1
	shmHeaderSize = 64
)

// shmLayout describes the memory layout of the shared buffer.
type shmLayout struct {
	numEnvs         int
	spatialChannels int
	spatialSize     int
	numScalars      int

	spatialOffset int
	scalarsOffset int
	masksOffset   int
	rewardsOffset int
	donesOffset   int
	actionsOffset int
	totalSize     int
}

func newSHMLayout(numEnvs int) shmLayout {
	ch := spatialChannels
	sz := spatialSize
	sc := numScalars
	n := numEnvs

	spatialBytes := n * ch * sz * sz * 4 // float32
	scalarsBytes := n * sc * 4            // float32
	masksBytes := n * numActions          // uint8
	rewardsBytes := n * 4                 // float32
	donesBytes := n                       // uint8

	off := shmHeaderSize
	spatialOff := off
	off += spatialBytes
	scalarsOff := off
	off += scalarsBytes
	masksOff := off
	off += masksBytes
	rewardsOff := off
	off += rewardsBytes
	donesOff := off
	off += donesBytes
	// Pad to 4-byte alignment for actions.
	off = (off + 3) &^ 3
	actionsOff := off
	off += n * 4 // int32

	return shmLayout{
		numEnvs:         n,
		spatialChannels: ch,
		spatialSize:     sz,
		numScalars:      sc,
		spatialOffset:   spatialOff,
		scalarsOffset:   scalarsOff,
		masksOffset:     masksOff,
		rewardsOffset:   rewardsOff,
		donesOffset:     donesOff,
		actionsOffset:   actionsOff,
		totalSize:       off,
	}
}

// shmBuffer holds the mmap'd shared memory region.
type shmBuffer struct {
	file   *os.File
	data   []byte
	layout shmLayout
	path   string
}

// createSHM creates the shared memory file and maps it.
func createSHM(path string, numEnvs int) (*shmBuffer, error) {
	layout := newSHMLayout(numEnvs)

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("shm create: %w", err)
	}
	if err := f.Truncate(int64(layout.totalSize)); err != nil {
		f.Close()
		os.Remove(path)
		return nil, fmt.Errorf("shm truncate: %w", err)
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, layout.totalSize,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		os.Remove(path)
		return nil, fmt.Errorf("shm mmap: %w", err)
	}

	buf := &shmBuffer{
		file:   f,
		data:   data,
		layout: layout,
		path:   path,
	}

	// Write header.
	copy(buf.data[0:4], []byte(shmMagic))
	binary.LittleEndian.PutUint32(buf.data[4:8], shmVersion)
	binary.LittleEndian.PutUint32(buf.data[8:12], uint32(numEnvs))
	binary.LittleEndian.PutUint32(buf.data[12:16], uint32(spatialChannels))
	binary.LittleEndian.PutUint32(buf.data[16:20], uint32(spatialSize))
	binary.LittleEndian.PutUint32(buf.data[20:24], uint32(numScalars))
	binary.LittleEndian.PutUint32(buf.data[24:28], uint32(layout.spatialOffset))
	binary.LittleEndian.PutUint32(buf.data[28:32], uint32(layout.scalarsOffset))
	binary.LittleEndian.PutUint32(buf.data[32:36], uint32(layout.masksOffset))
	binary.LittleEndian.PutUint32(buf.data[36:40], uint32(layout.rewardsOffset))
	binary.LittleEndian.PutUint32(buf.data[40:44], uint32(layout.donesOffset))
	binary.LittleEndian.PutUint32(buf.data[44:48], uint32(layout.actionsOffset))
	// step_counter at 48-55 starts at 0 (already zeroed).
	// reserved 56-63 already zeroed.

	return buf, nil
}

// closeSHM unmaps, closes, and removes the shared memory file.
func (b *shmBuffer) closeSHM() {
	if b.data != nil {
		syscall.Munmap(b.data)
		b.data = nil
	}
	if b.file != nil {
		b.file.Close()
		b.file = nil
	}
	os.Remove(b.path)
}

// writeSpatial copies spatial observation for one env into mmap.
func (b *shmBuffer) writeSpatial(envIdx int, spatial []float32) {
	floatsPerEnv := b.layout.spatialChannels * b.layout.spatialSize * b.layout.spatialSize
	off := b.layout.spatialOffset + envIdx*floatsPerEnv*4
	dst := unsafe.Slice((*float32)(unsafe.Pointer(&b.data[off])), floatsPerEnv)
	copy(dst, spatial)
}

// writeScalars copies scalar observation for one env into mmap.
func (b *shmBuffer) writeScalars(envIdx int, scalars []float32) {
	off := b.layout.scalarsOffset + envIdx*b.layout.numScalars*4
	dst := unsafe.Slice((*float32)(unsafe.Pointer(&b.data[off])), b.layout.numScalars)
	copy(dst, scalars)
}

// writeMask writes the action mask for one env into mmap.
func (b *shmBuffer) writeMask(envIdx int, mask [numActions]bool) {
	off := b.layout.masksOffset + envIdx*numActions
	for i := 0; i < numActions; i++ {
		if mask[i] {
			b.data[off+i] = 1
		} else {
			b.data[off+i] = 0
		}
	}
}

// writeReward writes the reward for one env into mmap.
func (b *shmBuffer) writeReward(envIdx int, reward float32) {
	off := b.layout.rewardsOffset + envIdx*4
	binary.LittleEndian.PutUint32(b.data[off:off+4], math.Float32bits(reward))
}

// writeDone writes the done flag for one env into mmap.
func (b *shmBuffer) writeDone(envIdx int, done bool) {
	off := b.layout.donesOffset + envIdx
	if done {
		b.data[off] = 1
	} else {
		b.data[off] = 0
	}
}

// readAction reads the action for one env from mmap.
func (b *shmBuffer) readAction(envIdx int) int32 {
	off := b.layout.actionsOffset + envIdx*4
	return int32(binary.LittleEndian.Uint32(b.data[off : off+4]))
}
