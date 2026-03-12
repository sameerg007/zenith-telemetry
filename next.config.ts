import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Do not expose "X-Powered-By: Next.js" response header.
  poweredByHeader: false,
  compress: true,
};

export default nextConfig;
