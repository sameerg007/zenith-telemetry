// Example API simulation for Go and Python benchmark data
export async function fetchGoBenchmarks(instrumentCount: number) {
  const response = await fetch(`http://127.0.0.1:8080/benchmark?count=${instrumentCount}`);
  if (!response.ok) {
    throw new Error('Failed to fetch Go benchmarks');
  }
  return response.json();
}

export async function fetchPythonBenchmarks(instrumentCount: number) {
  const response = await fetch(`http://127.0.0.1:8000/benchmark?count=${instrumentCount}`);
  if (!response.ok) {
    throw new Error('Failed to fetch Python benchmarks');
  }
  return response.json();
}
