import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  transpilePackages: ["smart-plan-demo"],
};

export default nextConfig;
