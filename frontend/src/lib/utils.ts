import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function splitList(value: string): string[] {
  return value.split(",").map((s) => s.trim()).filter(Boolean);
}
