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

const GO_API_URL = process.env.NEXT_PUBLIC_GO_API_URL ?? 'http://127.0.0.1:8080';
const PY_API_URL = process.env.NEXT_PUBLIC_PY_API_URL ?? 'http://127.0.0.1:8000';

async function apiFetch<T>(url: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(url, { signal });
  if (!response.ok) {
    throw new Error(`Request to ${new URL(url).host} failed with status ${response.status}`);
  }
  return response.json() as Promise<T>;
}

export function fetchGoBenchmarks(
  instrumentCount: number,
  signal?: AbortSignal,
): Promise<BenchmarkResult<GoMeasurement>> {
  return apiFetch<BenchmarkResult<GoMeasurement>>(
    `${GO_API_URL}/benchmark?count=${instrumentCount}`,
    signal,
  );
}

export function fetchPythonBenchmarks(
  instrumentCount: number,
  signal?: AbortSignal,
): Promise<BenchmarkResult<PyMeasurement>> {
  return apiFetch<BenchmarkResult<PyMeasurement>>(
    `${PY_API_URL}/benchmark?count=${instrumentCount}`,
    signal,
  );
}

