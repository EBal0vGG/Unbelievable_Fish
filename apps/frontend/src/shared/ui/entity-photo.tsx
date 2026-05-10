"use client";

import { useState, type ImgHTMLAttributes } from "react";

import { cn } from "@/shared/lib/cn";

interface EntityPhotoProps extends Omit<ImgHTMLAttributes<HTMLImageElement>, "src"> {
  src?: string | null;
  label?: string;
}

export function EntityPhoto({
  src,
  alt,
  label = "",
  className,
  ...props
}: EntityPhotoProps) {
  const [hasError, setHasError] = useState(false);
  const normalizedSrc = src?.trim();
  const fallbackLabel = alt?.trim()?.slice(0, 2).toUpperCase() || "UF";

  if (!normalizedSrc || hasError) {
    return (
      <div className={cn("entity-photo entity-photo-empty", className)}>
        <span>{fallbackLabel}</span>
      </div>
    );
  }

  return (
    <div className={cn("entity-photo", className)}>
      <img
        alt={alt}
        loading="lazy"
        src={normalizedSrc}
        onError={() => setHasError(true)}
        {...props}
      />
      {label ? <span>{label}</span> : null}
    </div>
  );
}
