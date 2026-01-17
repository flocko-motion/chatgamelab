import { useEffect, useCallback } from 'react';
import { IconX } from '@tabler/icons-react';
import { useGamePlayerContext } from '../context';
import classes from './GamePlayer.module.css';

export function ImageLightbox() {
  const { lightboxImage, closeLightbox } = useGamePlayerContext();

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      closeLightbox();
    }
  }, [closeLightbox]);

  useEffect(() => {
    if (lightboxImage) {
      document.addEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'hidden';
      return () => {
        document.removeEventListener('keydown', handleKeyDown);
        document.body.style.overflow = '';
      };
    }
  }, [lightboxImage, handleKeyDown]);

  if (!lightboxImage) return null;

  return (
    <div 
      className={classes.lightboxOverlay}
      onClick={closeLightbox}
      role="dialog"
      aria-modal="true"
      aria-label={lightboxImage.alt || 'Image preview'}
    >
      <button
        className={classes.lightboxClose}
        onClick={closeLightbox}
        aria-label="Close"
      >
        <IconX size={24} />
      </button>
      <img
        src={lightboxImage.url}
        alt={lightboxImage.alt || 'Scene illustration'}
        className={classes.lightboxImage}
        onClick={(e) => e.stopPropagation()}
      />
    </div>
  );
}
