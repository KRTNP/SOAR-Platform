import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Lean production image for the Dockerfile — see web/Dockerfile.
  output: "standalone",
};

export default nextConfig;
