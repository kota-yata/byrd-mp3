package maindata

import (
	"byrd/internal/common"
	"fmt"
)

var huffmanZeroTable = common.HuffmanTable{
	Data: []uint16{0x0000},
}

func selectTable(sampleRate uint16, gc *common.GranuleChannelInfo, spectralLineIndex int) (*common.HuffmanTable, error) {
	if gc == nil {
		return nil, fmt.Errorf("nil granule channel info")
	}
	if spectralLineIndex < 0 {
		return nil, fmt.Errorf("invalid spectral line index: %d", spectralLineIndex)
	}

	sfBands, ok := common.SCALEFACTOR_BAND_INDICES[sampleRate]
	if !ok {
		return nil, fmt.Errorf("unsupported sample rate for scalefactor bands: %d", sampleRate)
	}

	var tableIndex int
	if !gc.GetWindowSwitching() || gc.GetBlockType() != common.BlockTypeShort {
		region1StartSFB := int(gc.Region0Count) + 1
		region2StartSFB := int(gc.Region0Count) + int(gc.Region1Count) + 2
		if region1StartSFB >= len(sfBands.Long) {
			region1StartSFB = len(sfBands.Long) - 1
		}
		if region2StartSFB >= len(sfBands.Long) {
			region2StartSFB = len(sfBands.Long) - 1
		}
		region1Start := sfBands.Long[region1StartSFB]
		region2Start := sfBands.Long[region2StartSFB]
		if spectralLineIndex < region1Start {
			tableIndex = int(gc.TableSelect[0])
		} else if spectralLineIndex < region2Start {
			tableIndex = int(gc.TableSelect[1])
		} else {
			tableIndex = int(gc.TableSelect[2])
		}
	} else {
		// Short-block Huffman regions are used only for pure/mixed short blocks.
		// Start/end blocks keep long-block region boundaries even though
		// window_switching is set.
		region1Start := sfBands.Short[3] * SCALEFACTOR_SHORT_WINDOW_COUNT
		if gc.GetMixedBlockFlag() {
			region1Start = sfBands.Long[8]
		}
		if spectralLineIndex < region1Start {
			tableIndex = int(gc.TableSelect[0])
		} else {
			tableIndex = int(gc.TableSelect[1])
		}
	}

	if tableIndex == 0 {
		return &huffmanZeroTable, nil
	}

	table, ok := common.BaseTables[tableIndex]
	if !ok {
		return nil, fmt.Errorf("unknown huffman table: %d", tableIndex)
	}
	if table.Data == nil {
		return nil, fmt.Errorf("unsupported huffman table: %d", tableIndex)
	}
	return &table, nil
}

func guardedReadBit(br *common.BitReader, limit int, scratch *uint32) error {
	if br.Pos+1 > limit {
		return fmt.Errorf("huffman data exceeds part23 length: need 1 more bit, have %d", limit-br.Pos)
	}
	return br.ReadBitsTo(scratch, 1)
}

func guardedReadBits(br *common.BitReader, limit int, n int, scratch *uint32) error {
	if br.Pos+n > limit {
		return fmt.Errorf("huffman data exceeds part23 length: need %d more bits, have %d", n, limit-br.Pos)
	}
	return br.ReadBitsTo(scratch, n)
}

func isHuffmanLeaf(v uint16) bool {
	return v&0xFF00 == 0
}

func decodeHuffmanPair(br *common.BitReader, table *common.HuffmanTable, limit int, scratch *uint32) (int, int, error) {
	if table == nil {
		return 0, 0, fmt.Errorf("nil huffman table")
	}
	if len(table.Data) == 0 {
		return 0, 0, fmt.Errorf("empty huffman table")
	}
	treeLen := table.TreeLen
	if treeLen == 0 {
		treeLen = len(table.Data)
	}

	idx := 0
	for {
		if idx < 0 || idx >= len(table.Data) {
			return 0, 0, fmt.Errorf("invalid huffman tree traversal")
		}
		node := table.Data[idx]
		if isHuffmanLeaf(node) {
			x := int((node >> 4) & 0xF)
			y := int(node & 0xF)
			if table.Linbits > 0 {
				if x == 15 {
					if err := guardedReadBits(br, limit, table.Linbits, scratch); err != nil {
						return 0, 0, err
					}
					x += int(*scratch)
				}
				if x != 0 {
					if err := guardedReadBit(br, limit, scratch); err != nil {
						return 0, 0, err
					}
					if *scratch == 1 {
						x = -x
					}
				}
				if y == 15 {
					if err := guardedReadBits(br, limit, table.Linbits, scratch); err != nil {
						return 0, 0, err
					}
					y += int(*scratch)
				}
				if y != 0 {
					if err := guardedReadBit(br, limit, scratch); err != nil {
						return 0, 0, err
					}
					if *scratch == 1 {
						y = -y
					}
				}
				return x, y, nil
			}
			if x != 0 {
				if err := guardedReadBit(br, limit, scratch); err != nil {
					return 0, 0, err
				}
				if *scratch == 1 {
					x = -x
				}
			}
			if y != 0 {
				if err := guardedReadBit(br, limit, scratch); err != nil {
					return 0, 0, err
				}
				if *scratch == 1 {
					y = -y
				}
			}
			return x, y, nil
		}

		if err := guardedReadBit(br, limit, scratch); err != nil {
			return 0, 0, err
		}
		if *scratch != 0 {
			for (table.Data[idx] & 0x00FF) >= 250 {
				idx += int(table.Data[idx] & 0x00FF)
				if idx < 0 || idx >= len(table.Data) {
					return 0, 0, fmt.Errorf("invalid huffman tree traversal")
				}
			}
			idx += int(table.Data[idx] & 0x00FF)
		} else {
			for (table.Data[idx] >> 8) >= 250 {
				idx += int(table.Data[idx] >> 8)
				if idx < 0 || idx >= len(table.Data) {
					return 0, 0, fmt.Errorf("invalid huffman tree traversal")
				}
			}
			idx += int(table.Data[idx] >> 8)
		}
		if idx >= treeLen {
			return 0, 0, fmt.Errorf("invalid huffman tree traversal")
		}
	}
}

func decodeHuffmanQuad(br *common.BitReader, table *common.HuffmanTable, limit int, scratch *uint32) (int, int, int, int, error) {
	if table == nil {
		return 0, 0, 0, 0, fmt.Errorf("nil huffman table")
	}
	if len(table.Data) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("empty huffman table")
	}
	treeLen := table.TreeLen
	if treeLen == 0 {
		treeLen = len(table.Data)
	}

	idx := 0
	for {
		if idx < 0 || idx >= len(table.Data) {
			return 0, 0, 0, 0, fmt.Errorf("invalid huffman tree traversal")
		}
		node := table.Data[idx]
		if isHuffmanLeaf(node) {
			v := int((node >> 3) & 0x1)
			w := int((node >> 2) & 0x1)
			x := int((node >> 1) & 0x1)
			y := int(node & 0x1)
			values := []*int{&v, &w, &x, &y}
			for _, value := range values {
				if *value == 0 {
					continue
				}
				if err := guardedReadBit(br, limit, scratch); err != nil {
					return 0, 0, 0, 0, err
				}
				if *scratch == 1 {
					*value = -*value
				}
			}
			return v, w, x, y, nil
		}

		if err := guardedReadBit(br, limit, scratch); err != nil {
			return 0, 0, 0, 0, err
		}
		if *scratch != 0 {
			for (table.Data[idx] & 0x00FF) >= 250 {
				idx += int(table.Data[idx] & 0x00FF)
				if idx < 0 || idx >= len(table.Data) {
					return 0, 0, 0, 0, fmt.Errorf("invalid huffman tree traversal")
				}
			}
			idx += int(table.Data[idx] & 0x00FF)
		} else {
			for (table.Data[idx] >> 8) >= 250 {
				idx += int(table.Data[idx] >> 8)
				if idx < 0 || idx >= len(table.Data) {
					return 0, 0, 0, 0, fmt.Errorf("invalid huffman tree traversal")
				}
			}
			idx += int(table.Data[idx] >> 8)
		}
		if idx >= treeLen {
			return 0, 0, 0, 0, fmt.Errorf("invalid huffman tree traversal")
		}
	}
}
