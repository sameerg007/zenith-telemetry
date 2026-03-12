import { redirect } from 'next/navigation';

// The root of the app is the benchmark dashboard.
export default function Home() {
  redirect('/benchmark');
}
