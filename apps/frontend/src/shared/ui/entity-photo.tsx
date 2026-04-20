import type { ImgHTMLAttributes } from "react";

import { cn } from "@/shared/lib/cn";

interface EntityPhotoProps extends Omit<ImgHTMLAttributes<HTMLImageElement>, "src"> {
  src?: string | null;
  label?: string;
}

export function EntityPhoto({
  src,
  alt,
  label = "Фото поставки",
  className,
  ...props
}: EntityPhotoProps) {
  const normalizedSrc = src?.trim();

  if (!normalizedSrc) {
    return (
      <div className={cn("entity-photo entity-photo-empty", className)}>
        <span>{label}</span>
      </div>
    );
  }

  return (
    <div className={cn("entity-photo", className)}>
      <img
        alt={alt}
        loading="lazy"
        src={normalizedSrc}
        onError={(event) => {
          event.currentTarget.style.display = "none";
          event.currentTarget.parentElement?.classList.add("entity-photo-empty");
        }}
        {...props}
      />
      <span>{label}</span>
    </div>
  );
}
