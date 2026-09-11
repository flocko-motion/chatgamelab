import { useEffect, useCallback, useState } from 'react';
import { IconX } from '@tabler/icons-react';
import { useGamePlayerContext } from '../context';
import classes from './GamePlayer.module.css';

type LightboxImage = { url: string; alt?: string };

export function ImageLightbox() {
  const { lightboxImage, closeLightbox } = useGamePlayerContext();

  // Keep the last shown image mounted while the lightbox is closed, so the
  // <img> element and its decoded bitmap survive close/re-open (instant re-open,
  // no re-fetch, no re-decode). This is the React "store info from previous
  // renders" pattern: adjust state during render, not in an effect.
  const [shown, setShown] = useState<LightboxImage | null>(lightboxImage);
  const [seen, setSeen] = useState(lightboxImage);
  if (lightboxImage && lightboxImage !== seen) {
    setSeen(lightboxImage);
    setShown(lightboxImage);
  }

  const open = !!lightboxImage;

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      closeLightbox();
    }
  }, [closeLightbox]);

  useEffect(() => {
    if (open) {
      document.addEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'hidden';
      return () => {
        document.removeEventListener('keydown', handleKeyDown);
        document.body.style.overflow = '';
      };
    }
  }, [open, handleKeyDown]);

  if (!shown) return null;

  return (
    <div
      className={classes.lightboxOverlay}
      style={{ display: open ? undefined : 'none' }}
      onClick={closeLightbox}
      role="dialog"
      aria-modal="true"
      aria-hidden={!open}
      aria-label={shown.alt || 'Image preview'}
    >
      <button
        className={classes.lightboxClose}
        onClick={closeLightbox}
        aria-label="Close"
        tabIndex={open ? 0 : -1}
      >
        <IconX size={24} />
      </button>
      <img
        src={shown.url}
        alt={shown.alt || 'Scene illustration'}
        className={classes.lightboxImage}
        onClick={(e) => e.stopPropagation()}
      />
    </div>
  );
}
