
"use client";
import React, { useState, useEffect, useCallback } from 'react';
import { AgGridReact } from 'ag-grid-react';
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community';
import type { ColDef, CellClassParams, CellStyle } from 'ag-grid-community';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-quartz.css';
import { Toaster, toast } from 'react-hot-toast';
import { fetchGoBenchmarks, fetchPythonBenchmarks } from './benchmarkApi';
import { BENCHMARK_CONFIG } from './config';

ModuleRegistry.registerModules([AllCommunityModule]);

type Row = {
  device: string;
  goLatency: string;
  pyLatency: string;
  goData: string;
  pyData: string;
  result: string;
};

// Defined outside the component so the reference is stable across renders.
const COLUMN_DEFS: ColDef<Row>[] = [
  { headerName: 'Device', field: 'device', minWidth: 100 },
  {
    headerName: 'Go Latency (ms)',
    field: 'goLatency',
    minWidth: 150,
    cellStyle: { backgroundColor: '#e0f7fa', color: '#006064', fontWeight: 'bold' },
  },
  {
    headerName: 'Python Latency (ms)',
    field: 'pyLatency',
    minWidth: 170,
    cellStyle: { backgroundColor: '#fff3e0', color: '#bf360c', fontWeight: 'bold' },
  },
  {
    headerName: 'Go Data',
    field: 'goData',
    minWidth: 130,
    cellStyle: { backgroundColor: '#e0f7fa', color: '#006064' },
  },
  {
    headerName: 'Python Data',
    field: 'pyData',
    minWidth: 130,
    cellStyle: { backgroundColor: '#fff3e0', color: '#bf360c' },
  },
  {
    headerName: 'Winner',
    field: 'result',
    minWidth: 100,
    cellStyle: (params: CellClassParams<Row, string>): CellStyle | null => {
      if (params.value === 'Go') return { backgroundColor: '#b2dfdb', color: '#004d40', fontWeight: 'bold' };
      if (params.value === 'Python') return { backgroundColor: '#ffe0b2', color: '#e65100', fontWeight: 'bold' };
      return null;
    },
  },
];

function computeSummary(goCycleTime: string, pyCycleTime: string): string {
  const goCycle = parseFloat(goCycleTime);
  const pyCycle = parseFloat(pyCycleTime);
  if (isNaN(goCycle) || isNaN(pyCycle)) return '';
  if (goCycle < pyCycle) return `Go is faster (cycle ${goCycle.toFixed(3)} ms vs ${pyCycle.toFixed(3)} ms)`;
  if (pyCycle < goCycle) return `Python is faster (cycle ${pyCycle.toFixed(3)} ms vs ${goCycle.toFixed(3)} ms)`;
  return `Both engines tied (cycle ${goCycle.toFixed(3)} ms)`;
}

export default function BenchmarkPage() {
  const { defaultInstrumentCount, minInstruments, maxInstruments, debounceMs } = BENCHMARK_CONFIG;

  const [instrumentCount, setInstrumentCount] = useState<number>(defaultInstrumentCount);
  const [debouncedCount, setDebouncedCount] = useState<number>(defaultInstrumentCount);
  const [rows, setRows] = useState<Row[]>([]);
  const [summary, setSummary] = useState<string>('');
  const [goCycleTime, setGoCycleTime] = useState<string>('');
  const [pyCycleTime, setPyCycleTime] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  // Debounce instrument count so rapid input changes don't fire a fetch per keystroke.
  useEffect(() => {
    const timer = setTimeout(() => {
      const count = Math.max(minInstruments, Math.min(maxInstruments, instrumentCount));
      setDebouncedCount(count);
    }, debounceMs);
    return () => clearTimeout(timer);
  }, [instrumentCount, debounceMs, minInstruments, maxInstruments]);

  useEffect(() => {
    const controller = new AbortController();
    const { signal } = controller;

    const fetchData = async () => {
      setLoading(true);
      setError(null);
      try {
        const [goResp, pyResp] = await Promise.all([
          fetchGoBenchmarks(debouncedCount, signal),
          fetchPythonBenchmarks(debouncedCount, signal),
        ]);

        if (signal.aborted) return;

        const goMeasurements = goResp.measurements ?? [];
        const pyMeasurements = pyResp.measurements ?? [];

        const goCycle = goResp.cycleTimeMs ?? '';
        const pyCycle = pyResp.cycleTimeMs ?? '';
        setGoCycleTime(goCycle);
        setPyCycleTime(pyCycle);

        // Determine the overall winner from cycle time (total batch wall-clock).
        // Per-row individual latencies are dominated by the same mock I/O delay
        // and do not reflect the parallel engine's throughput advantage.
        const goCycleMs = parseFloat(goCycle);
        const pyCycleMs = parseFloat(pyCycle);
        let overallWinner = 'N/A';
        if (!isNaN(goCycleMs) && !isNaN(pyCycleMs)) {
          if (goCycleMs < pyCycleMs) overallWinner = 'Go';
          else if (pyCycleMs < goCycleMs) overallWinner = 'Python';
          else overallWinner = 'Equal';
        }

        const merged: Row[] = goMeasurements.map((g, i) => {
          const py = pyMeasurements[i] ?? { device: g.device, pyLatency: '', pyData: '' };
          return {
            device: g.device,
            goLatency: g.goLatency,
            pyLatency: py.pyLatency,
            goData: g.goData,
            pyData: py.pyData,
            result: overallWinner,
          };
        });

        setRows(merged);
        setSummary(computeSummary(goCycle, pyCycle));
      } catch (err) {
        if ((err as Error).name === 'AbortError') return;
        console.error('Benchmark fetch failed:', err);
        setError(
          'Could not reach backend servers. Make sure the Go engine (port 8080) and Python engine (port 8000) are running.',
        );
      } finally {
        if (!signal.aborted) setLoading(false);
      }
    };

    fetchData();
    return () => controller.abort();
  }, [debouncedCount]);

  const handleCountChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const val = parseInt(e.target.value, 10);
    if (isNaN(val)) {
      setInstrumentCount(minInstruments);
      return;
    }
    if (val > maxInstruments) {
      toast.error(`Max allowed instruments is ${maxInstruments}`);
    }
    setInstrumentCount(val);
  }, [maxInstruments, minInstruments]);

  return (
    <div className="p-8">
      <Toaster position="top-center" />
      <h1 className="text-2xl font-bold mb-1">Zenith Benchmark — Parallel Engine Comparison</h1>
      <p className="text-sm text-gray-500 mb-4">
        Go: goroutines + channels · Python: ThreadPoolExecutor — both poll real mock TCP instruments (SCPI over TCP)
      </p>

      <div className="mb-4 flex items-center gap-4 flex-wrap">
        <label htmlFor="instrument-count" className="font-medium">
          Instruments:
        </label>
        <input
          id="instrument-count"
          type="number"
          value={instrumentCount}
          onChange={handleCountChange}
          className="border px-2 py-1 rounded w-20"
          aria-label="Number of instruments to benchmark"
        />
        {goCycleTime && (
          <span
            className="text-sm px-3 py-1 rounded bg-cyan-50 text-cyan-800 font-mono"
            aria-label={`Go cycle time: ${goCycleTime} milliseconds`}
          >
            Go cycle: {goCycleTime} ms
          </span>
        )}
        {pyCycleTime && (
          <span
            className="text-sm px-3 py-1 rounded bg-orange-50 text-orange-800 font-mono"
            aria-label={`Python cycle time: ${pyCycleTime} milliseconds`}
          >
            Python cycle: {pyCycleTime} ms
          </span>
        )}
      </div>

      {loading && (
        <div className="mb-2 text-blue-600" role="status" aria-live="polite">
          Running parallel benchmark…
        </div>
      )}
      {error && (
        <div className="mb-2 text-red-600" role="alert">
          {error}
        </div>
      )}

      <div className="ag-theme-quartz" style={{ height: 400, width: '100%' }}>
        <AgGridReact<Row> theme="legacy" rowData={rows} columnDefs={COLUMN_DEFS} />
      </div>

      {summary && !loading && (
        <div className="mt-6 text-lg font-semibold" role="status" aria-live="polite">
          Result: {summary}
        </div>
      )}
    </div>
  );
}


