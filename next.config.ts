import type { NextConfig } from "next";

const securityHeaders = [
  // Prevent MIME-type sniffing (stops browsers executing non-JS as scripts).
  { key: 'X-Content-Type-Options', value: 'nosniff' },
  // Deny framing to mitigate clickjacking.
  { key: 'X-Frame-Options', value: 'DENY' },
  // Tell browsers to use strict HTTPS for 1 year (applies once served over TLS).
  { key: 'Strict-Transport-Security', value: 'max-age=31536000; includeSubDomains' },
  // Limit referrer data sent to third parties.
  { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
  // Restrict access to sensitive browser APIs this app does not need.
  { key: 'Permissions-Policy', value: 'camera=(), microphone=(), geolocation=()' },
];

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Do not expose "X-Powered-By: Next.js" response header.
  poweredByHeader: false,
  compress: true,
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: securityHeaders,
      },
    ];
  },
};

export default nextConfig;
