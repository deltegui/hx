package hx

import "log"

// keyedRow is a row of an EachKeyed list. It is built once and lives while its
// key is in the list.
type keyedRow[T any] struct {
	node    INode
	item    *SignalT[T]
	index   *SignalT[int]
	dispose func()
}

// EachKeyed renders a list where every row is identified by a key. Unlike Each,
// the rows are not rebuilt when the list changes: a row is built once, the first
// time its key appears, and then it is reused, moved if the order changed,
// and removed when its key disappears. Because the DOM elements are kept, inputs
// keep their value and focus, scroll positions stay and nothing flickers.
//
// key returns the identity of an item, it has to be unique in the list.
// render builds the row. It receives the key, and the item and its position as
// values that change over time (read them with Get inside a binding or an
// effect, they are not read once). Effects, computed values and bindings created
// inside render belong to the row and are stopped when the row is removed.
//
// Unlike Each, EachKeyed follows the source by itself, so use it directly as a
// child instead of inside an effect:
//
//	hx.Ul().Body(hx.EachKeyed(todos,
//		func(t Todo) int { return t.ID },
//		func(id int, todo hx.Gettable[Todo], index hx.Gettable[int]) hx.INode {
//			return hx.Li().BindText(hx.Computed(func() string { return todo.Get().Title }))
//		}))
//
// If two items have the same key a warning is logged and the later one is
// rendered as a row of its own that is rebuilt every time.
func EachKeyed[T any, K comparable](
	src Gettable[[]T],
	key func(item T) K,
	render func(key K, item Gettable[T], index Gettable[int]) INode,
) INode {
	list := Noop()
	rows := map[K]*keyedRow[T]{}
	var duplicates []*keyedRow[T]

	build := func(k K, item T, i int) *keyedRow[T] {
		row := &keyedRow[T]{item: Signal(item), index: Signal(i)}
		// Outside of the effect that is reconciling, so what render creates is not
		// disposed when the list changes, and inside a root of its own, so it can
		// be stopped when the row goes away.
		Untrack(func() {
			row.dispose = CreateRoot(func(dispose func()) func() {
				row.node = render(k, row.item, row.index)
				return dispose
			})
		})
		return row
	}

	reconcile := func() {
		items := src.Get()
		for _, row := range duplicates {
			row.dispose()
		}
		duplicates = nil

		next := make(map[K]*keyedRow[T], len(items))
		nodes := make([]INode, 0, len(items))
		for i, item := range items {
			k := key(item)
			if _, repeated := next[k]; repeated {
				log.Printf("EachKeyed: duplicated key %v, the row is rebuilt every time", k)
				row := build(k, item, i)
				duplicates = append(duplicates, row)
				if row.node != nil {
					nodes = append(nodes, row.node)
				}
				continue
			}
			row, ok := rows[k]
			if ok {
				row.item.Set(item)
				if row.index.value != i {
					row.index.Set(i)
				}
			} else {
				row = build(k, item, i)
			}
			next[k] = row
			if row.node != nil {
				nodes = append(nodes, row.node)
			}
		}
		for k, row := range rows {
			if _, kept := next[k]; !kept {
				row.dispose()
			}
		}
		rows = next
		list.BodyList(nodes)
	}

	stop := func() {
		for _, row := range rows {
			row.dispose()
		}
		for _, row := range duplicates {
			row.dispose()
		}
		rows = map[K]*keyedRow[T]{}
		duplicates = nil
	}

	if b, ok := list.(binder); ok {
		b.BindWith(reconcile, stop)
	} else {
		list.Effect(reconcile)
	}
	return list
}
