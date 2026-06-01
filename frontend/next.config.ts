import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  reactStrictMode: true,
  turbopack: {
    root: __dirname,
  },

    allowedDevOrigins: [
    'localhost',
    '127.0.0.1', 
    '10.9.39.68', 
  ],
};

export default nextConfig;
