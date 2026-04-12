import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  /* config options here */
  reactStrictMode: true,

    allowedDevOrigins: [
    'localhost',
    '127.0.0.1', 
    '192.168.0.21', 
  ],
};

export default nextConfig;
