import { createCanvas, loadImage, GlobalFonts } from '@napi-rs/canvas';
import { spawnSync } from 'node:child_process';
import { writeFile, copyFile } from 'node:fs/promises';
import { resolve, dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const out = resolve(process.argv[2] || join(root, 'artifacts/promo'));
const frame = join(out, 'work/poster-frame.jpg');
const result = spawnSync('ffmpeg', ['-hide_banner','-loglevel','error','-y','-ss','2','-i',join(out,'ltseries-generated-sample.mp4'),'-frames:v','1',frame], {encoding:'utf8'});
if(result.status !== 0) throw new Error(result.stderr);
GlobalFonts.registerFromPath('/System/Library/Fonts/Avenir Next.ttc', 'Poster');
const canvas = createCanvas(1920,1080), ctx=canvas.getContext('2d');
ctx.fillStyle='#111217';ctx.fillRect(0,0,1920,1080);
function text(value,x,y,size,color='#f4f5f7',weight=400){ctx.font=`${weight} ${size}px Poster, sans-serif`;ctx.fillStyle=color;ctx.fillText(value,x,y);}
text('LTSeries',68,156,118,'#f4f5f7',700);
text('Script. Assets. Storyboard. Film.',74,226,40);
text('OPEN SOURCE  /  SELF-HOSTED',1290,90,25,'#bbef76',600);
text('Go + React + MongoDB',1290,135,28,'#a4a9b7');
ctx.fillStyle='#bbef76';ctx.beginPath();ctx.moveTo(1300,178);ctx.lineTo(1300,218);ctx.lineTo(1334,198);ctx.closePath();ctx.fill();
text('WATCH THE DEMO',1352,209,29,'#bbef76',600);
text('REAL WORKSPACE',74,288,22,'#a4a9b7',600);
text('GENERATED SAMPLE',1420,288,22,'#a4a9b7',600);
ctx.drawImage(await loadImage(join(root,'docs/images/assets.jpg')),70,310,1280,720);
ctx.drawImage(await loadImage(frame),1420,310,405,720);
const cover=join(out,'cover-landscape.jpg');
await writeFile(cover,canvas.toBuffer('image/jpeg',90));
await copyFile(cover,join(root,'docs/images/demo-cover.jpg'));
console.log('Created cover and README poster from real screenshots and film frames.');
