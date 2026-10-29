package hx

import (
	"fmt"
	"testing"
)

type keyedItem struct {
	id   int
	name string
}

type keyedHarness struct {
	items   *SignalT[[]keyedItem]
	list    INode
	ul      INode
	built   map[int]int
	nodes   map[int]INode
	effects map[int]int // runs of an effect created inside render
	side    *SignalT[int]
}

func newKeyedHarness(items ...keyedItem) *keyedHarness {
	h := &keyedHarness{
		items:   Signal(items),
		built:   map[int]int{},
		nodes:   map[int]INode{},
		effects: map[int]int{},
		side:    Signal(0),
	}
	h.list = EachKeyed(h.items,
		func(it keyedItem) int { return it.id },
		func(id int, it Gettable[keyedItem], index Gettable[int]) INode {
			h.built[id]++
			EffectFunc(func() { h.effects[id]++; h.side.Get() })
			n := Li().BindText(Computed(func() string {
				return fmt.Sprintf("%d:%s", index.Get(), it.Get().name)
			}))
			h.nodes[id] = n
			return n
		})
	h.ul = Ul().Body(h.list)
	return h
}

func TestEachKeyedRendersAndUpdatesRows(t *testing.T) {
	h := newKeyedHarness(keyedItem{1, "a"}, keyedItem{2, "b"})
	if got := render(h.ul); got != `<ul><li>0:a</li><li>1:b</li></ul>` {
		t.Fatalf("got %s", got)
	}

	// reorder, change a value and add a row
	h.items.Set([]keyedItem{{2, "B"}, {3, "c"}, {1, "a"}})
	if got := render(h.ul); got != `<ul><li>0:B</li><li>1:c</li><li>2:a</li></ul>` {
		t.Fatalf("got %s", got)
	}
}

func TestEachKeyedReusesTheNodesOfExistingKeys(t *testing.T) {
	h := newKeyedHarness(keyedItem{1, "a"}, keyedItem{2, "b"})
	first, second := h.nodes[1], h.nodes[2]

	h.items.Set([]keyedItem{{2, "b"}, {1, "a"}, {3, "c"}})

	if h.nodes[1] != first || h.nodes[2] != second {
		t.Fatal("rows of existing keys must be the same nodes")
	}
	if h.built[1] != 1 || h.built[2] != 1 || h.built[3] != 1 {
		t.Fatalf("each row must be built once, built=%v", h.built)
	}
}

func TestEachKeyedStopsTheEffectsOfRemovedRows(t *testing.T) {
	h := newKeyedHarness(keyedItem{1, "a"}, keyedItem{2, "b"})
	h.side.Set(1)
	if h.effects[1] != 2 || h.effects[2] != 2 {
		t.Fatalf("effects=%v", h.effects)
	}

	h.items.Set([]keyedItem{{2, "b"}})
	h.side.Set(2)
	if h.effects[1] != 2 {
		t.Fatalf("the effect of a removed row kept running: %v", h.effects)
	}
	if h.effects[2] != 3 {
		t.Fatalf("the effect of a kept row must keep running: %v", h.effects)
	}
	if got := render(h.ul); got != `<ul><li>0:b</li></ul>` {
		t.Fatalf("got %s", got)
	}
}

func TestEachKeyedRebuildsARemovedAndReturnedKey(t *testing.T) {
	h := newKeyedHarness(keyedItem{1, "a"})
	h.items.Set([]keyedItem{})
	h.items.Set([]keyedItem{{1, "a"}})
	if h.built[1] != 2 {
		t.Fatalf("a key that went away and came back is a new row, built=%v", h.built)
	}
}

func TestEachKeyedStopsWhenTheListLeavesTheTree(t *testing.T) {
	h := newKeyedHarness(keyedItem{1, "a"})
	h.ul.deactivate()

	h.items.Set([]keyedItem{{1, "a"}, {2, "b"}})
	if h.built[2] != 0 {
		t.Fatal("a deactivated list must not reconcile")
	}
	h.side.Set(1)
	if h.effects[1] != 1 {
		t.Fatalf("row effects must be stopped, effects=%v", h.effects)
	}

	h.ul.reactivate()
	if got := render(h.ul); got != `<ul><li>0:a</li><li>1:b</li></ul>` {
		t.Fatalf("got %s", got)
	}
}

func TestEachKeyedDuplicatedKeysAreStillRendered(t *testing.T) {
	h := newKeyedHarness(keyedItem{1, "a"}, keyedItem{1, "again"})
	if got := render(h.ul); got != `<ul><li>0:a</li><li>1:again</li></ul>` {
		t.Fatalf("got %s", got)
	}
}

func TestEachKeyedSkipsNilRows(t *testing.T) {
	items := Signal([]int{1, 2, 3})
	list := EachKeyed(items, func(i int) int { return i }, func(k int, it Gettable[int], idx Gettable[int]) INode {
		if k == 2 {
			return nil
		}
		return Li().Text(fmt.Sprint(k))
	})
	if got := render(Ul().Body(list)); got != `<ul><li>1</li><li>3</li></ul>` {
		t.Fatalf("got %s", got)
	}
}
