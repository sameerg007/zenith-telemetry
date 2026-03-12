/** Shared UI constants for the Zenith benchmark page. */
export const BENCHMARK_CONFIG = {
  defaultInstrumentCount: 5,
  minInstruments: 1,
  maxInstruments: 50,
  /** Milliseconds to wait after the last input change before firing a fetch. */
  debounceMs: 400,
} as const;
