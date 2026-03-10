
  "use client";
import React, { useState, useEffect } from 'react';
import { AgGridReact } from 'ag-grid-react';
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-quartz.css';
import { fetchGoBenchmarks, fetchPythonBenchmarks } from './benchmarkApi';

ModuleRegistry.registerModules([AllCommunityModule]);

type Row = { device: string; goLatency: string; pyLatency: string; goData: string; pyData: string; result: string };

export default function BenchmarkPage() {
  const defaultInstrumentCount = 5;

  const columns = [
    { headerName: 'Device', field: 'device' },
    {
      headerName: 'Go Latency (ms)',
      field: 'goLatency',
      cellStyle: () => ({ backgroundColor: '#e0f7fa', color: '#006064', fontWeight: 'bold' })
    },
    {
      headerName: 'Python Latency (ms)',
      field: 'pyLatency',
      cellStyle: () => ({ backgroundColor: '#fff3e0', color: '#bf360c', fontWeight: 'bold' })
    },
    {
      headerName: 'Go Data',
      field: 'goData',
      cellStyle: () => ({ backgroundColor: '#e0f7fa', color: '#006064' })
    },
    {
      headerName: 'Python Data',
      field: 'pyData',
      cellStyle: () => ({ backgroundColor: '#fff3e0', color: '#bf360c' })
    },
    {
      headerName: 'Winner',
      field: 'result',
      cellStyle: (params: any) => {
        if (params.value === 'Go') return { backgroundColor: '#b2dfdb', color: '#004d40', fontWeight: 'bold' };
        if (params.value === 'Python') return { backgroundColor: '#ffe0b2', color: '#e65100', fontWeight: 'bold' };
        return {};
      }
    }
  ];

  const [instrumentCount, setInstrumentCount] = useState<number>(defaultInstrumentCount);
  const [rows, setRows] = useState<Row[]>([]);
  const [result, setResult] = useState<string>('');
  const [goCycleTime, setGoCycleTime] = useState<string>('');
  const [pyCycleTime, setPyCycleTime] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      setError(null);
      try {
        const [goResp, pyResp] = await Promise.all([
          fetchGoBenchmarks(instrumentCount),
          fetchPythonBenchmarks(instrumentCount),
        ]);

        const goMeasurements = goResp.measurements ?? [];
        const pyMeasurements = pyResp.measurements ?? [];

        setGoCycleTime(goResp.cycleTimeMs ?? '');
        setPyCycleTime(pyResp.cycleTimeMs ?? '');

        const merged: Row[] = goMeasurements.map((g, i) => {
          const py = pyMeasurements[i] ?? { device: '', pyLatency: '', pyData: '' };
          const goLatency = parseFloat(g.goLatency);
          const pyLatency = parseFloat(py.pyLatency);
          let winner = '';
          if (!isNaN(goLatency) && !isNaN(pyLatency)) {
            if (goLatency < pyLatency) winner = 'Go';
            else if (pyLatency < goLatency) winner = 'Python';
            else winner = 'Equal';
          }
          return { device: g.device, goLatency: g.goLatency, pyLatency: py.pyLatency, goData: g.goData, pyData: py.pyData, result: winner };
        });

        setRows(merged);

        const avgGo = merged.reduce((s, r) => s + (parseFloat(r.goLatency) || 0), 0) / merged.length;
        const avgPy = merged.reduce((s, r) => s + (parseFloat(r.pyLatency) || 0), 0) / merged.length;
        setResult(avgGo < avgPy ? `Go faster (avg ${avgGo.toFixed(3)} ms vs ${avgPy.toFixed(3)} ms)` : `Python faster (avg ${avgPy.toFixed(3)} ms vs ${avgGo.toFixed(3)} ms)`);
      } catch (err) {
        console.error("Failed to fetch benchmark data", err);
        setError(`Failed to load data: ${(err as Error).message}. Run backend/go (port 8080) and backend/python (port 8000) first.`);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [instrumentCount]);

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold mb-1">Zenith Benchmark — Parallel Engine Comparison</h1>
      <p className="text-sm text-gray-500 mb-4">
        Go: goroutines + channels · Python: threading — both poll real mock TCP instruments (SCPI over TCP)
      </p>

      <div className="mb-4 flex items-center gap-4">
        <label className="font-medium">Instruments:</label>
        <input
          type="number"
          min={1}
          max={50}
          value={instrumentCount}
          onChange={e => setInstrumentCount(Math.max(1, Math.min(50, Number(e.target.value))))}
          className="border px-2 py-1 rounded w-20"
        />
        {goCycleTime && (
          <span className="text-sm px-3 py-1 rounded bg-cyan-50 text-cyan-800 font-mono">
            Go cycle: {goCycleTime} ms
          </span>
        )}
        {pyCycleTime && (
          <span className="text-sm px-3 py-1 rounded bg-orange-50 text-orange-800 font-mono">
            Python cycle: {pyCycleTime} ms
          </span>
        )}
      </div>

      {loading && <div className="mb-2 text-blue-600">Running parallel benchmark…</div>}
      {error && <div className="mb-2 text-red-600">{error}</div>}

      <div className="ag-theme-quartz" style={{ height: 400, width: '100%' }}>
        <AgGridReact<Row>
          theme="legacy"
          rowData={rows}
          columnDefs={columns as any}
        />
      </div>

      {result && !loading && (
        <div className="mt-6 text-lg font-semibold">
          Result: {result}
        </div>
      )}
    </div>
  );
}
