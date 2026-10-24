import { createCanvas, loadImage, GlobalFonts } from '@napi-rs/canvas';
import { spawnSync } from 'node:child_process';
import { mkdir, readFile, writeFile, access } from 'node:fs/promises';
import { dirname, resolve, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseArgs } from 'node:util';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const { values: args } = parseArgs({ options: {
  sample: { type: 'string' }, output: { type: 'string' },
  font: { type: 'string' }, voice: { type: 'string' },
} });
if (!args.sample) throw new Error('Pass --sample /absolute/path/to/real-generated-video.mp4');
const sample = resolve(args.sample);
await access(sample);
const out = resolve(args.output || join(root, 'artifacts/promo'));
const scratch = join(out, 'work');
await mkdir(scratch, { recursive: true });
const scenes = JSON.parse(await readFile(join(root, 'docs/media/storyboard.json'), 'utf8'));
const fontPath = args.font || '/System/Library/Fonts/Hiragino Sans GB.ttc';
await access(fontPath);
GlobalFonts.registerFromPath(fontPath, 'Promo');

function run(command, argv) {
  const result = spawnSync(command, argv, { encoding: 'utf8', maxBuffer: 10 * 1024 * 1024 });
  if (result.status !== 0) throw new Error(`${command} failed: ${result.stderr || result.error}`);
  return result.stdout;
}
const ffmpeg = argv => run('ffmpeg', ['-hide_banner', '-loglevel', 'error', '-y', ...argv]);
const probe = file => JSON.parse(run('ffprobe', ['-v', 'error', '-show_entries', 'format=duration,size:stream=codec_name,width,height', '-of', 'json', file]));
const bg = '#111217', white = '#f4f5f7', muted = '#a4a9b7', accent = '#bbef76';
function text(ctx, value, x, y, size, color = white, weight = 400) {
  ctx.font = `${weight} ${size}px Promo, sans-serif`;
  ctx.fillStyle = color;
  ctx.fillText(value, x, y);
}
function wrap(ctx, value, x, y, width, size, lineHeight, color = white) {
  ctx.font = `400 ${size}px Promo, sans-serif`;
  let line = '', lines = 0;
  for (const ch of value) {
    if (ctx.measureText(line + ch).width > width && line) {
      text(ctx, line, x, y + lines++ * lineHeight, size, color);
      line = ch;
    } else line += ch;
  }
  if (line) text(ctx, line, x, y + lines++ * lineHeight, size, color);
  return lines * lineHeight;
}
function imageContain(ctx, img, x, y, width, height, crop = false) {
  if (crop) {
    // Focus the real main pane for vertical exports, retaining the full screenshot in the landscape edition.
    const left = Math.min(256, img.width * .2);
    const ratio = Math.min(width / (img.width - left), height / img.height);
    const w = (img.width - left) * ratio, h = img.height * ratio;
    ctx.drawImage(img, left, 0, img.width - left, img.height, x + (width-w)/2, y+(height-h)/2, w,h);
  } else {
    const ratio = Math.min(width / img.width, height / img.height);
    const w = img.width * ratio, h = img.height * ratio;
    ctx.drawImage(img, x+(width-w)/2, y+(height-h)/2,w,h);
  }
}
function clock(t) {
  const ms = Math.round(t * 1000);
  return `${String(Math.floor(ms/3600000)).padStart(2,'0')}:${String(Math.floor(ms/60000)%60).padStart(2,'0')}:${String(Math.floor(ms/1000)%60).padStart(2,'0')},${String(ms%1000).padStart(3,'0')}`;
}

let position = 0, srt = '', enSrt = '';
for (let i=0;i<scenes.length;i++) {
  const scene = scenes[i];
  scene.audio = join(scratch, `${i}.aiff`);
  // Narration is post-production TTS, not an in-app audio-generation demonstration.
  run('say', ['-v', args.voice || 'Tingting', '-r', '195', '-o', scene.audio, scene.voice]);
  scene.duration = Math.max(5, Math.ceil(Number(probe(scene.audio).format.duration) + .6));
  srt += `${i+1}\n${clock(position)} --> ${clock(position+scene.duration)}\n${scene.voice}\n\n`;
  enSrt += `${i+1}\n${clock(position)} --> ${clock(position+scene.duration)}\n${scene.translation}\n\n`;
  position += scene.duration;
}
await writeFile(join(out,'ltseries-zh.srt'), srt);
await writeFile(join(out,'ltseries-en.srt'), enSrt);

for (const format of ['landscape','portrait']) {
  const portrait = format === 'portrait';
  const W = portrait ? 1080 : 1920, H = portrait ? 1920 : 1080;
  const cuts = [];
  for(let i=0;i<scenes.length;i++) {
    const scene = scenes[i];
    const canvas = createCanvas(W,H), ctx = canvas.getContext('2d');
    ctx.fillStyle=bg; ctx.fillRect(0,0,W,H);
    ctx.fillStyle=accent;ctx.fillRect(56,64,6,35);
    text(ctx,'LTSeries',80,95,34,white,700);
    text(ctx,portrait?'开源 AI 短剧工作台':'SELF-HOSTED AI PRODUCTION STUDIO',portrait?80:1270,portrait?145:92,portrait?25:20,muted);
    const videoBox = portrait ? {x:170,y:440,w:740,h:1120} : {x:1110,y:170,w:470,h:770};
    if (portrait) {
      text(ctx,`${String(i+1).padStart(2,'0')} / ${String(scenes.length).padStart(2,'0')}`,56,230,25,accent);
      wrap(ctx,scene.title,56,310,W-112,60,82);
      if(scene.image) imageContain(ctx, await loadImage(join(root,scene.image)),28,540,W-56,850,true);
      if(scene.video) {ctx.clearRect(videoBox.x,videoBox.y,videoBox.w,videoBox.h);}
      text(ctx,scene.label,56,1640,25,accent);
      wrap(ctx,scene.caption,56,1700,W-112,34,52);
      text(ctx,'github.com/eust-w/ltseries',56,1830,27,muted);
    } else {
      text(ctx,`${String(i+1).padStart(2,'0')} / ${String(scenes.length).padStart(2,'0')}`,56,225,25,accent);
      wrap(ctx,scene.title,56,315,380,52,76);
      wrap(ctx,scene.caption,56,590,360,27,46,muted);
      text(ctx,scene.label,56,875,20,accent);
      if(scene.image) imageContain(ctx,await loadImage(join(root,scene.image)),470,150,1390,782);
      if(scene.video) ctx.clearRect(videoBox.x,videoBox.y,videoBox.w,videoBox.h);
      text(ctx,'github.com/eust-w/ltseries',56,1020,25,muted);
      text(ctx,scene.english,470,990,28,white);
    }
    const plate=join(scratch,`${format}-${i}.png`);
    await writeFile(plate,canvas.toBuffer('image/png'));
    const target=join(scratch,`${format}-${i}.mp4`);
    const inputs=['-loop','1','-framerate','30','-i',plate];
    let filter;
    if(scene.video) {
      inputs.push('-stream_loop','-1','-ss',String(scene.offset||0),'-i',sample,'-i',scene.audio);
      filter=`[1:v]scale=${videoBox.w}:${videoBox.h}:force_original_aspect_ratio=decrease,pad=${videoBox.w}:${videoBox.h}:(ow-iw)/2:(oh-ih)/2:color=${bg.replace('#','0x')},setsar=1[v];[0:v][v]overlay=${videoBox.x}:${videoBox.y}:shortest=1,fade=t=in:st=0:d=0.2,fade=t=out:st=${scene.duration-.2}:d=0.2[out]`;
    } else {
      inputs.push('-i',scene.audio);
      filter=`[0:v]fade=t=in:st=0:d=0.2,fade=t=out:st=${scene.duration-.2}:d=0.2[out]`;
    }
    ffmpeg([...inputs,'-filter_complex',filter,'-map','[out]','-map',scene.video?'2:a':'1:a','-af','apad,loudnorm=I=-16:TP=-1.5:LRA=9','-t',String(scene.duration),'-r','30','-c:v','libx264','-preset','fast','-crf','21','-pix_fmt','yuv420p','-c:a','aac','-b:a','160k','-ar','48000','-ac','2',target]);
    cuts.push(target);
    console.log(`Rendered ${format} ${i+1}/${scenes.length}`);
  }
  const list=join(scratch,`${format}.txt`);
  await writeFile(list,cuts.map(p=>`file '${p.replaceAll("'", "'\\''")}'`).join('\n'));
  const final=join(out,`ltseries-${format}-zh.mp4`);
  ffmpeg(['-f','concat','-safe','0','-i',list,'-c','copy','-movflags','+faststart',final]);
  ffmpeg(['-ss','2','-i',final,'-frames:v','1',join(out,`cover-${format}.jpg`)]);
}

// Animated excerpts use genuine captured pages and the existing generated film, not synthetic UI.
ffmpeg(['-ss','0','-i',sample,'-t','6','-filter_complex','fps=8,scale=270:-1:flags=lanczos,split[a][b];[a]palettegen=stats_mode=diff[p];[b][p]paletteuse=dither=bayer:bayer_scale=3','-loop','0',join(root,'docs/media/generated-film.gif')]);
const uiCuts=join(scratch,'ui-tour.txt');
await writeFile(uiCuts,scenes.filter(s=>s.image).map(s=>`file '${join(root,s.image)}'\nduration 2`).join('\n')+`\nfile '${join(root,scenes.filter(s=>s.image).at(-1).image)}'\n`);
ffmpeg(['-f','concat','-safe','0','-i',uiCuts,'-filter_complex','fps=5,scale=800:-1:flags=lanczos,split[a][b];[a]palettegen=max_colors=128[p];[b][p]paletteuse=dither=bayer:bayer_scale=4','-loop','0',join(root,'docs/media/workflow-tour.gif')]);
ffmpeg(['-i',sample,'-an','-c:v','libx264','-crf','22','-pix_fmt','yuv420p','-movflags','+faststart',join(out,'ltseries-generated-sample.mp4')]);
const manifest={created:new Date().toISOString(),duration_seconds:position,narration:'macOS Tingting synthetic voice, added in post-production',sample:'Historical real LTSeries output; not regenerated during this recording',files:[]};
for(const file of ['ltseries-landscape-zh.mp4','ltseries-portrait-zh.mp4','ltseries-generated-sample.mp4'])manifest.files.push({name:file,...probe(join(out,file))});
await writeFile(join(out,'verification.json'),JSON.stringify(manifest,null,2)+'\n');
console.log(`Finished: ${out}`);
