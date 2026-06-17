import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  experimental: {
    externalDir: true,
  },
  transpilePackages: ["smart-plan-demo"],
};

export default nextConfig;
