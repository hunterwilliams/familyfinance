const MAX_EDGE = 1920;
const TARGET_BYTES = 900_000;
const MAX_SOURCE_BYTES = 12 * 1024 * 1024;

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

export async function prepareReceiptImage(file: File): Promise<{ file: File; note: string }> {
  if (file.size > MAX_SOURCE_BYTES) {
    throw new Error(`Photo is ${formatBytes(file.size)}. Use an image under 12 MB.`);
  }

  const decoded = await decodeImage(file);
  const { width, height } = decoded;
  const needsResize = Math.max(width, height) > MAX_EDGE || file.size > TARGET_BYTES;
  if (!needsResize) {
    decoded.close();
    return { file, note: '' };
  }

  const scale = Math.min(1, MAX_EDGE / Math.max(width, height));
  const w = Math.max(1, Math.round(width * scale));
  const h = Math.max(1, Math.round(height * scale));
  const canvas = document.createElement('canvas');
  canvas.width = w;
  canvas.height = h;
  const ctx = canvas.getContext('2d');
  if (!ctx) {
    decoded.close();
    throw new Error('Could not process that photo in this browser.');
  }
  ctx.drawImage(decoded.source, 0, 0, w, h);
  decoded.close();

  const qualities = [0.85, 0.75, 0.65, 0.55];
  let blob: Blob | null = null;
  for (const q of qualities) {
    blob = await canvasToJpeg(canvas, q);
    if (blob.size <= TARGET_BYTES) break;
  }
  if (!blob) {
    throw new Error('Could not compress that photo. Try a JPEG screenshot.');
  }

  const next = new File([blob], jpegName(file.name), { type: 'image/jpeg' });
  const note = `Reduced ${formatBytes(file.size)} → ${formatBytes(next.size)}`;
  return { file: next, note };
}

function jpegName(name: string): string {
  const base = name.replace(/\.[^.]+$/, '') || 'receipt';
  return `${base}.jpg`;
}

function canvasToJpeg(canvas: HTMLCanvasElement, quality: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (blob) resolve(blob);
        else reject(new Error('Could not encode JPEG'));
      },
      'image/jpeg',
      quality,
    );
  });
}

type Decoded = {
  source: CanvasImageSource;
  width: number;
  height: number;
  close: () => void;
};

async function decodeImage(file: File): Promise<Decoded> {
  if (typeof createImageBitmap === 'function') {
    try {
      const bitmap = await createImageBitmap(file);
      return {
        source: bitmap,
        width: bitmap.width,
        height: bitmap.height,
        close: () => bitmap.close(),
      };
    } catch {
      // HEIC / odd types: fall through to HTMLImageElement
    }
  }

  const url = URL.createObjectURL(file);
  try {
    const img = await loadImage(url);
    return {
      source: img,
      width: img.naturalWidth,
      height: img.naturalHeight,
      close: () => URL.revokeObjectURL(url),
    };
  } catch {
    URL.revokeObjectURL(url);
    throw new Error('Could not read that photo. Try JPEG or PNG, or take a new picture.');
  }
}

function loadImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error('decode'));
    img.src = url;
  });
}
