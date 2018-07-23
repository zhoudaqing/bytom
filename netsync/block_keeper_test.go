package netsync

import (
	"container/list"
	"testing"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/test/mock"
	"github.com/bytom/testutil"
)

func TestAppendHeaderList(t *testing.T) {
	blocks := mockBlocks(7)
	cases := []struct {
		originalHeaders []*types.BlockHeader
		inputHeaders    []*types.BlockHeader
		wantHeaders     []*types.BlockHeader
		err             error
	}{
		{
			originalHeaders: []*types.BlockHeader{&blocks[0].BlockHeader},
			inputHeaders:    []*types.BlockHeader{&blocks[1].BlockHeader, &blocks[2].BlockHeader},
			wantHeaders:     []*types.BlockHeader{&blocks[0].BlockHeader, &blocks[1].BlockHeader, &blocks[2].BlockHeader},
			err:             nil,
		},
		{
			originalHeaders: []*types.BlockHeader{&blocks[5].BlockHeader},
			inputHeaders:    []*types.BlockHeader{&blocks[6].BlockHeader},
			wantHeaders:     []*types.BlockHeader{&blocks[5].BlockHeader, &blocks[6].BlockHeader},
			err:             nil,
		},
		{
			originalHeaders: []*types.BlockHeader{&blocks[5].BlockHeader},
			inputHeaders:    []*types.BlockHeader{&blocks[7].BlockHeader},
			wantHeaders:     []*types.BlockHeader{&blocks[5].BlockHeader},
			err:             errAppendHeaders,
		},
		{
			originalHeaders: []*types.BlockHeader{&blocks[5].BlockHeader},
			inputHeaders:    []*types.BlockHeader{&blocks[7].BlockHeader, &blocks[6].BlockHeader},
			wantHeaders:     []*types.BlockHeader{&blocks[5].BlockHeader},
			err:             errAppendHeaders,
		},
		{
			originalHeaders: []*types.BlockHeader{&blocks[2].BlockHeader},
			inputHeaders:    []*types.BlockHeader{&blocks[3].BlockHeader, &blocks[4].BlockHeader, &blocks[6].BlockHeader},
			wantHeaders:     []*types.BlockHeader{&blocks[2].BlockHeader, &blocks[3].BlockHeader, &blocks[4].BlockHeader},
			err:             errAppendHeaders,
		},
	}

	for i, c := range cases {
		bk := &blockKeeper{headerList: list.New()}
		for _, header := range c.originalHeaders {
			bk.headerList.PushBack(header)
		}

		if err := bk.appendHeaderList(c.inputHeaders); err != c.err {
			t.Errorf("case %d: got error %v want error %v", i, err, c.err)
		}

		gotHeaders := []*types.BlockHeader{}
		for e := bk.headerList.Front(); e != nil; e = e.Next() {
			gotHeaders = append(gotHeaders, e.Value.(*types.BlockHeader))
		}

		if !testutil.DeepEqual(gotHeaders, c.wantHeaders) {
			t.Errorf("case %d: got %v want %v", i, gotHeaders, c.wantHeaders)
		}
	}
}

func TestBlockLocator(t *testing.T) {
	blocks := mockBlocks(500)
	cases := []struct {
		bestHeight uint64
		wantHeight []uint64
	}{
		{
			bestHeight: 0,
			wantHeight: []uint64{0},
		},
		{
			bestHeight: 1,
			wantHeight: []uint64{1, 0},
		},
		{
			bestHeight: 7,
			wantHeight: []uint64{7, 6, 5, 4, 3, 2, 1, 0},
		},
		{
			bestHeight: 10,
			wantHeight: []uint64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		},
		{
			bestHeight: 100,
			wantHeight: []uint64{100, 99, 98, 97, 96, 95, 94, 93, 92, 91, 89, 85, 77, 61, 29, 0},
		},
		{
			bestHeight: 500,
			wantHeight: []uint64{500, 499, 498, 497, 496, 495, 494, 493, 492, 491, 489, 485, 477, 461, 429, 365, 237, 0},
		},
	}

	for i, c := range cases {
		mockChain := mock.NewChain()
		bk := &blockKeeper{chain: mockChain}
		mockChain.SetBestBlockHeader(&blocks[c.bestHeight].BlockHeader)
		for i := uint64(0); i <= c.bestHeight; i++ {
			mockChain.SetBlockByHeight(i, blocks[i])
		}

		want := []*bc.Hash{}
		for _, i := range c.wantHeight {
			hash := blocks[i].Hash()
			want = append(want, &hash)
		}

		if got := bk.blockLocator(); !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func TestLocateBlocks(t *testing.T) {
	maxBlockPerMsg = 5
	blocks := mockBlocks(100)
	cases := []struct {
		locator    []uint64
		stopHash   bc.Hash
		wantHeight []uint64
	}{
		{
			locator:    []uint64{20},
			stopHash:   blocks[100].Hash(),
			wantHeight: []uint64{21, 22, 23, 24, 25},
		},
	}

	mockChain := mock.NewChain()
	bk := &blockKeeper{chain: mockChain}
	for _, block := range blocks {
		mockChain.SetBlockByHeight(block.Height, block)
	}

	for i, c := range cases {
		locator := []*bc.Hash{}
		for _, i := range c.locator {
			hash := blocks[i].Hash()
			locator = append(locator, &hash)
		}

		want := []*types.Block{}
		for _, i := range c.wantHeight {
			want = append(want, blocks[i])
		}

		got, _ := bk.locateBlocks(locator, &c.stopHash)
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func TestLocateHeaders(t *testing.T) {
	maxBlockHeadersPerMsg = 10
	blocks := mockBlocks(150)
	cases := []struct {
		chainHeight uint64
		locator     []uint64
		stopHash    bc.Hash
		wantHeight  []uint64
		err         bool
	}{
		{
			chainHeight: 100,
			locator:     []uint64{},
			stopHash:    blocks[100].Hash(),
			wantHeight:  []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    blocks[100].Hash(),
			wantHeight:  []uint64{21, 22, 23, 24, 25, 26, 27, 28, 29, 30},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    blocks[24].Hash(),
			wantHeight:  []uint64{21, 22, 23, 24},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    blocks[20].Hash(),
			wantHeight:  []uint64{},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    bc.Hash{},
			wantHeight:  []uint64{},
			err:         true,
		},
		{
			chainHeight: 100,
			locator:     []uint64{120, 70},
			stopHash:    blocks[78].Hash(),
			wantHeight:  []uint64{71, 72, 73, 74, 75, 76, 77, 78},
			err:         false,
		},
	}

	for i, c := range cases {
		mockChain := mock.NewChain()
		bk := &blockKeeper{chain: mockChain}
		for i := uint64(0); i <= c.chainHeight; i++ {
			mockChain.SetBlockByHeight(i, blocks[i])
		}

		locator := []*bc.Hash{}
		for _, i := range c.locator {
			hash := blocks[i].Hash()
			locator = append(locator, &hash)
		}

		want := []*types.BlockHeader{}
		for _, i := range c.wantHeight {
			want = append(want, &blocks[i].BlockHeader)
		}

		got, err := bk.locateHeaders(locator, &c.stopHash)
		if err != nil != c.err {
			t.Errorf("case %d: got %v want err = %v", i, err, c.err)
		}
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func mockBlocks(height uint64) []*types.Block {
	blocks := []*types.Block{}
	preHash := bc.Hash{}
	for i := uint64(0); i <= height; i++ {
		block := &types.Block{
			BlockHeader: types.BlockHeader{
				Height:            i,
				PreviousBlockHash: preHash,
			},
		}
		blocks = append(blocks, block)
		preHash = block.Hash()
	}
	return blocks
}
