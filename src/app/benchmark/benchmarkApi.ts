export interface BenchmarkResult<T> {
  cycleTimeMs: string;
  measurements: T[];
}

export interface GoMeasurement {
  device: string;
  goLatency: string;
  goData: string;
}

export interface PyMeasurement {
  device: string;
  pyLatency: string;
  pyData: string;
}

export async function fetchGoBenchmarks(instrumentCount: number): Promise<BenchmarkResult<GoMeasurement>> {
  const response = await fetch(`http://127.0.0.1:8080/benchmark?count=${instrumentCount}`);
  if (!response.ok) {
    throw new Error('Failed to fetch Go benchmarks');
  }
  return response.json();
}

export async function fetchPythonBenchmarks(instrumentCount: number): Promise<BenchmarkResult<PyMeasurement>> {
  const response = await fetch(`http://127.0.0.1:8000/benchmark?count=${instrumentCount}`);
  if (!response.ok) {
    throw new Error('Failed to fetch Python benchmarks');
  }
  return response.json();
}
