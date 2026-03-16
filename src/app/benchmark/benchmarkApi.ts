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

/** Hard client-side timeout per fetch. Prevents the UI from hanging indefinitely
 *  if a backend server stalls or is unreachable. */
const FETCH_TIMEOUT_MS = 15_000;

async function apiFetch<T>(url: string, signal?: AbortSignal): Promise<T> {
  // Create a local controller so we can apply a hard timeout.
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);

  // Forward any external abort (e.g. component unmount, count change) to our controller.
  signal?.addEventListener('abort', () => controller.abort(), { once: true });

  try {
    const response = await fetch(url, { signal: controller.signal });
    if (!response.ok) {
      throw new Error(`Request to ${new URL(url).host} failed with status ${response.status}`);
    }
    return response.json() as Promise<T>;
  } finally {
    clearTimeout(timeoutId);
  }
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

