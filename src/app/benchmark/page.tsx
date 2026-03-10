
  "use client";
import React, { useState, useEffect } from 'react';
import { AgGridReact } from 'ag-grid-react';
import { ModuleRegistry, AllCommunityModule } from 'ag-grid-community';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-quartz.css';
import { fetchGoBenchmarks, fetchPythonBenchmarks } from './benchmarkApi';

ModuleRegistry.registerModules([AllCommunityModule]);

export default function BenchmarkPage() {
  const defaultInstrumentCount = 5;

  const columns = [
    { headerName: 'Device', field: 'device' },
    {
      headerName: 'Go Latency (ms)',
      field: 'goLatency',
      cellStyle: (params: any) => ({ backgroundColor: '#e0f7fa', color: '#006064', fontWeight: 'bold' })
    },
    {
      headerName: 'Python Latency (ms)',
      field: 'pyLatency',
      cellStyle: (params: any) => ({ backgroundColor: '#fff3e0', color: '#bf360c', fontWeight: 'bold' })
    },
    {
      headerName: 'Go Data',
      field: 'goData',
      cellStyle: (params: any) => ({ backgroundColor: '#e0f7fa', color: '#006064' })
    },
    {
      headerName: 'Python Data',
      field: 'pyData',
      cellStyle: (params: any) => ({ backgroundColor: '#fff3e0', color: '#bf360c' })
    },
    {
      headerName: 'Result',
      field: 'result',
      cellStyle: (params: any) => {
        if (params.value === 'Go') return { backgroundColor: '#b2dfdb', color: '#004d40', fontWeight: 'bold' };
        if (params.value === 'Python') return { backgroundColor: '#ffe0b2', color: '#e65100', fontWeight: 'bold' };
        return {};
      }
    }
  ];

  const [instrumentCount, setInstrumentCount] = useState<number>(defaultInstrumentCount);
  const [rows, setRows] = useState<Array<{ device: string; goLatency: string; pyLatency: string; goData: string; pyData: string; result: string }>>([]);
  const [result, setResult] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  // Fetch benchmark results from API
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      setError(null);
      try {
        const goResults = await fetchGoBenchmarks(instrumentCount);
        const pyResults = await fetchPythonBenchmarks(instrumentCount);

        if (!Array.isArray(goResults) || !Array.isArray(pyResults)) {
          throw new Error("Invalid data format received from backend");
        }

        const merged = goResults.map((g: any, i: number) => {
          const py = pyResults[i] || {};
          const goLatency = parseFloat(g.goLatency);
          const pyLatency = parseFloat(py.pyLatency);
          let result = '';
          if (!isNaN(goLatency) && !isNaN(pyLatency)) {
            if (goLatency < pyLatency) result = 'Go';
            else if (pyLatency < goLatency) result = 'Python';
            else result = 'Equal';
          }
          return {
            device: g.device,
            goLatency: g.goLatency,
            pyLatency: py.pyLatency,
            goData: g.goData,
            pyData: py.pyData,
            result
          };
        });
        setRows(merged);
        // Simple comparison
        const avgGo = merged.reduce((sum: number, r: { goLatency: string }) => sum + (parseFloat(r.goLatency) || 0), 0) / instrumentCount;
        const avgPy = merged.reduce((sum: number, r: { pyLatency: string }) => sum + (parseFloat(r.pyLatency) || 0), 0) / instrumentCount;
        setResult(avgGo < avgPy ? 'Go performed better' : 'Python performed better');
      } catch (error) {
        console.error("Failed to fetch benchmark data", error);
        setError(`Failed to load data: ${(error as Error).message}. Ensure Go (port 8080) and Python (port 8000) servers are running.`);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [instrumentCount]);

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold mb-4">Zenith Benchmark Comparison</h1>
      <div className="mb-4">
        <label className="mr-2">Number of Instruments:</label>
        <input
          type="number"
          min={1}
          max={50}
          value={instrumentCount}
          onChange={e => {
            const newCount = Number(e.target.value);
            setInstrumentCount(newCount);
          }}
          className="border px-2 py-1 rounded"
        />
        {/* Run Benchmarks button removed, now auto-fetches */}
      </div>
      {loading && <div className="mb-2 text-blue-600">Loading benchmark data...</div>}
      {error && <div className="mb-2 text-red-600">{error}</div>}
      <div className="ag-theme-quartz" style={{ height: 400, width: '100%' }}>
        <AgGridReact<{ device: string; goLatency: string; pyLatency: string; goData: string; pyData: string; result: string }>
          theme="legacy"
          rowData={rows}
          columnDefs={columns as any}
        />
      </div>
      {result && (
        <div className="mt-6 text-lg font-semibold">
          Result: {result}
        </div>
      )}
    </div>
  );
}
