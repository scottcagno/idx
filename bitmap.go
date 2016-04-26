package idx

import "os"

const (
	bitmapSize = 1 << 19 // in bits
	wordSize   = 8       // bits / word
)

var bitmapTable = [16]byte{0, 1, 1, 2, 1, 2, 2, 3, 1, 2, 2, 3, 2, 3, 3, 4}

type BitMmap struct {
	path string
	file *os.File
	size int
	used int
	mmap Data
}

func OpenBitMmap(path string) *BitMmap {
	file, path, size := OpenFile(path + ".idx")
	if size == 0 {
		size = resize(file.Fd(), bitmapSize/wordSize)
	}

	bm := &BitMmap{}
	bm.path = path + ".idx"
	bm.file = file
	bm.size = size
	bm.mmap = Mmap(file, 0, size)
	bm.used = bm.Used()
	return bm
}

func (bm *BitMmap) Has(k int) bool {
	return (bm.mmap[k/wordSize] & (1 << (uint(k % wordSize)))) != 0
}

func (bm *BitMmap) Add() int {
	if k := bm.Next(); k != -1 {
		bm.Set(k) // add
		return k
	}
	return -1
}

func (bm *BitMmap) Set(k int) {
	// flip the n-th bit on; add/set
	bm.mmap[k/wordSize] |= (1 << uint(k%wordSize))
	bm.used++
}

func (bm *BitMmap) Del(k int) {
	// flip the k-th bit off; delete
	bm.mmap[k/wordSize] &= ^(1 << uint(k%wordSize))
	bm.used--
}

func (bm *BitMmap) bits(n byte) int {
	return int(bitmapTable[n>>4] + bitmapTable[n&0x0f])
}

// closes the mapped file
func (bm *BitMmap) CloseBitMmap() {
	bm.mmap.Sync()
	bm.mmap.Munmap()
	bm.file.Close()
}

func (bm *BitMmap) Next() int {
	bm.checkGrow()
	for i := 0; i < len(bm.mmap); i++ {
		if bm.bits(bm.mmap[i]) < 8 {
			for j := 0; j < 8; j++ {
				cur := (i * wordSize) + j
				if !bm.Has(cur) {
					return cur
				}
			}
		}
	}
	return -1
}

func (bm *BitMmap) Used() int {
	var used int
	for i := 0; i < bm.size; i++ {
		used += bm.bits(bm.mmap[i])
	}
	return used
}

func (bm *BitMmap) All() []int {
	var all []int
	for i := 0; i < len(bm.mmap); i++ {
		if bm.bits(bm.mmap[i]) <= 8 {
			for j := 0; j < 8; j++ {
				cur := (i * wordSize) + j
				if bm.Has(cur) {
					all = append(all, cur)
				}
			}
		}
	}
	return all
}

func (bm *BitMmap) checkGrow() {
	if bm.used+1 < bm.size*8 {
		return // no need to grow
	}
	bm.mmap.Munmap()
	bm.size = resize(bm.file.Fd(), bm.size+(bitmapSize/wordSize))
	bm.mmap = Mmap(bm.file, 0, bm.size)
}
