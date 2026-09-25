package core

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"sync/atomic"
)

// VMFallbackAttribution is opt-in: the normal call path does not maintain
// counters or maps. Sampling avoids a map update on every VM-to-AST call.
const vmFallbackSampleRate = 256

type vmFallbackSite struct {
	callee *FnExpr
	caller string
	line   int
}

type vmFallbackStats struct {
	calls atomic.Uint64
	mu    sync.Mutex
	sites map[vmFallbackSite]uint64
}

var vmFallbackAttribution atomic.Pointer[vmFallbackStats]

func StartVMFallbackAttribution() {
	vmFallbackAttribution.Store(&vmFallbackStats{sites: make(map[vmFallbackSite]uint64)})
}

func (stats *vmFallbackStats) record(fn *Fn, vm *VM) {
	if stats.calls.Add(1)%vmFallbackSampleRate != 0 {
		return
	}
	frame := &vm.frames[vm.frameCount-1]
	caller := "<top-level>"
	if frame.closure != nil && frame.closure.proto != nil {
		caller = frame.closure.proto.Name
	}
	// OP_CALL's opcode and argument-count byte have both been consumed.
	line := 0
	if ip := frame.ip - 2; ip >= 0 && ip < len(frame.arityProto.Chunk.Lines) {
		line = frame.arityProto.Chunk.Lines[ip]
	}
	stats.mu.Lock()
	stats.sites[vmFallbackSite{callee: fn.fnExpr, caller: caller, line: line}]++
	stats.mu.Unlock()
}

// WriteVMFallbackAttribution writes an approximate, call-count-ranked report.
// Each site is identified by callee definition and caller function/line.
func WriteVMFallbackAttribution(w io.Writer) {
	stats := vmFallbackAttribution.Swap(nil)
	if stats == nil {
		return
	}
	type entry struct {
		count uint64
		label string
	}
	stats.mu.Lock()
	entries := make([]entry, 0, len(stats.sites))
	for site, count := range stats.sites {
		pos := site.callee.Pos()
		calleeName := "<anonymous>"
		if site.callee.self.name != nil {
			calleeName = site.callee.self.Name()
		}
		caller := site.caller
		if site.line > 0 {
			caller = fmt.Sprintf("%s:%d", caller, site.line)
		}
		entries = append(entries, entry{
			count: count,
			label: fmt.Sprintf("%s -> %s %s:%d:%d", caller, calleeName, pos.Filename(), pos.startLine, pos.startColumn),
		})
	}
	stats.mu.Unlock()
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].label < entries[j].label
	})
	fmt.Fprintf(w, "VM->AST fallback calls: %d (sampled 1/%d; estimated calls per site)\n", stats.calls.Load(), vmFallbackSampleRate)
	for i, e := range entries {
		if i == 20 {
			break
		}
		fmt.Fprintf(w, "%12d  %s\n", e.count*vmFallbackSampleRate, e.label)
	}
}
