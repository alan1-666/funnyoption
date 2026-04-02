import type { NextConfig } from "next";

const distDir = process.env.NODE_ENV === "development" ? ".next-dev" : ".next";

const nextConfig: NextConfig = {
  distDir,
  experimental: {
    typedRoutes: true
  }
};

export default nextConfig;
