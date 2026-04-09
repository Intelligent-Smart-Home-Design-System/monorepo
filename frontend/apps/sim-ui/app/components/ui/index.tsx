"use client";

import type { ReactNode } from "react";

type Props = {
  children: ReactNode;
  className?: string;
};

export function Card({ children, className }: Props) {
  return <div className={["glass-card", className ?? ""].join(" ")}>{children}</div>;
}