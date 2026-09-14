# Offline static translation of the restored MPZ decoder into C.
# Usage: lift.py RESTORED_DLL OBJDUMP_TEXT OUTPUT_DIRECTORY
import re, collections, pathlib, struct, sys, hashlib
if len(sys.argv) != 4:
 raise SystemExit("usage: lift.py RESTORED_DLL OBJDUMP_TEXT OUTPUT_DIRECTORY")
dll, assembly, outdir = map(pathlib.Path, sys.argv[1:])
p = outdir
p.mkdir(parents=True, exist_ok=True)
expected = "e9a3db23a3c89f6a4ec2e59456bb85ae03f7ffde36b1c689c239fe38794f2e27"
if hashlib.sha256(dll.read_bytes()).hexdigest() != expected:
 raise SystemExit("unsupported MPZ decoder image")
lines=assembly.read_text().splitlines()
ins={}
for l in lines:
 m=re.match(r'^([0-9a-f]+):\s+(\S+)\s*(.*?)\s*$',l)
 if m:
  a=int(m[1],16)
  if 0x10001000<=a<0x1001938c:ins[a]=(m[2],m[3].split(' #')[0].split(' <')[0])
seq=sorted(ins); nxt=dict(zip(seq,seq[1:]))
# File callbacks are statically known; CRT is supplied by the memory stream adapter.
roots=[0x10016dd0,0x100133d0,0x10016c7c,0x10018e24,0x10018e10,0x100158cc]
funcs={}; external=set(); todo=roots.copy()
while todo:
 f=todo.pop()
 if f in funcs:continue
 seen=set(); q=[f]
 while q:
  a=q.pop()
  if a in seen:continue
  if a not in ins:raise ValueError(hex(a))
  seen.add(a); op,args=ins[a]
  if op=='calll' and not args.startswith('*'):
   dest=int(args,16)
   if dest in ins:todo.append(dest)
   else:external.add(dest)
  if op=='retl':continue
  if op.startswith('j'):
   q.append(int(args,16))
   if op=='jmp':continue
  q.append(nxt[a])
 funcs[f]=seen
# Flags are local to each translated routine. Calls have the x86 C ABI,
# which does not preserve arithmetic flags.
flag_after={}
for f,seen in funcs.items():
 live={a:0 for a in seen}
 changed=True
 while changed:
  changed=False
  for a in sorted(seen,reverse=True):
   op,args=ins[a]
   successors=[] if op=='retl' else [int(args,0)] if op=='jmp' else [nxt[a],int(args,0)] if op.startswith('j') else [nxt[a]]
   out=0
   for b in successors:out|=live[b]
   use=0; define=0
   if op.startswith('j') and op!='jmp' or op.startswith('set'):
    suffix=op[1:] if op.startswith('j') else op[3:]
    use={'e':1,'ne':1,'b':2,'ae':2,'a':3,'be':3,'l':12,'ge':12,'le':13,'g':13}[suffix]
   elif op.startswith(('add','sub','cmp','xor','or','and','test','neg','call','imul','mul','idiv','div')):define=15
   elif op.startswith(('sar','shr','shl')):
    # A variable shift count can be zero, preserving previous flags.
    if args.startswith('$') or len(split_args:=re.split(r',\s*(?![^()]*\))',args))==1:define=15
   elif op=='adcl':use=2;define=15
   value=use|(out&~define)
   flag_after[a]=flag_after.get(a,0)|out
   if value!=live[a]:live[a]=value;changed=True
 if live[f]:raise ValueError(('flags at routine entry',hex(f),live[f]))
reg32={'eax','ebx','ecx','edx','esi','edi','ebp','esp'}
regs={k:(k,32,0) for k in reg32}
for k in ('ax','bx','cx','dx','si','di','bp','sp'):regs[k]=('e'+k,16,0)
for k in ('a','b','c','d'):
 regs[k+'l']=('e'+k+'x',8,0); regs[k+'h']=('e'+k+'x',8,8)
def split(s):return [x.strip() for x in re.split(r',\s*(?![^()]*\))',s)] if s else []
def addr(x):
 if x.startswith('%fs:'):return '0'
 if '(' not in x:return str(int(x,0)&0xffffffff)+'u'
 m=re.fullmatch(r'([^()]*)\(([^()]*)\)',x); assert m,x
 disp,rs=m.groups(); rs=rs.split(','); out=[str(int(disp or '0',0)&0xffffffff)+'u']
 if rs[0]:out.append(rs[0][1:])
 if len(rs)>1 and rs[1]:out.append(rs[1][1:]+'*'+(rs[2] if len(rs)>2 else '1')+'u')
 return '('+'+'.join(out)+')'
def get(x,w=32):
 if x.startswith('$'):return str(int(x[1:],0)&0xffffffff)+'u'
 if x.startswith('%') and not x.startswith('%fs:'):
  k,b,shift=regs[x[1:]]
  return k if b==32 else f'(({k}>>{shift})&{(1<<b)-1}u)'
 if x.startswith('%fs:'):return 's->fs0'
 return f'rd{w}(s,{addr(x)})'
def put(x,v,w=32):
 if x.startswith('%') and not x.startswith('%fs:'):
  k,b,shift=regs[x[1:]]
  return f'{k}={v};' if b==32 else f'{k}=({k}&{0xffffffff^(((1<<b)-1)<<shift)}u)|((({v})&{(1<<b)-1}u)<<{shift});'
 if x.startswith('%fs:'):return f's->fs0={v};'
 return f'wr{w}(s,{addr(x)},{v});'
cond={'e':'s->zf','ne':'!s->zf','b':'s->cf','be':'(s->cf||s->zf)','a':'(!s->cf&&!s->zf)','ae':'!s->cf','l':'(s->sf!=s->of)','le':'(s->zf||s->sf!=s->of)','g':'(!s->zf&&s->sf==s->of)','ge':'(s->sf==s->of)'}
def emit(a,op,args):
 xs=split(args); w=8 if op.endswith('b') else 16 if op.endswith('w') else 32
 if op in ('movl','movw','movb'):return put(xs[1],get(xs[0],w),w)
 if op in ('movzbl','movzwl','movsbl','movswl'):
  b=8 if op[-2]=='b' else 16; v=get(xs[0],b)
  if op[3]=='s':v=f'(uint32_t)(int32_t)(int{b}_t)({v})'
  return put(xs[1],v)
 if op=='leal':return put(xs[1],addr(xs[0]))
 if op=='pushl':return f'push(s,{get(xs[0])});'
 if op=='popl':return 't=pop(s);'+put(xs[0],'t')
 if op=='retl':return f'esp+={4+(int(xs[0][1:],0) if xs else 0)}u; return;'
 if op=='calll':
  dest=args if not args.startswith('*') else get(args[1:])
  call=f'f_{int(args,0):x}(s)' if not args.startswith('*') and int(args,0) in funcs else f'callback(s,{dest})'
  return f'push(s,{nxt[a]}u); {call};'
 if op=='jmp':return f'goto L{int(args,0):x};'
 if op.startswith('j'):return f'if({cond[op[1:]]})goto L{int(args,0):x};'
 if op.startswith('set'):return put(xs[0],cond[op[3:]],8)
 if op in ('addl','addb','subl','cmpl','xorl','orl','andl','testl','testb','adcl'):
  f={'addl':'add','addb':'add','subl':'sub','cmpl':'sub','xorl':'xor','orl':'or','andl':'and','testl':'and','testb':'and','adcl':'adc'}[op]
  if not flag_after[a] and op!='adcl':
   if op.startswith(('cmp','test')):return ';'
   sym={'add':'+','sub':'-','xor':'^','or':'|','and':'&'}[f]
   return put(xs[1],f'({get(xs[1],w)}{sym}{get(xs[0],w)})',w)
  v=f'alu(s,{get(xs[1],w)},{get(xs[0],w)},{w},{dict(add=0,sub=1,xor=2,or_=3,and_=4,adc=5).get(f+"_",dict(add=0,sub=1,xor=2,adc=5).get(f))})'
  return f'(void){v};' if op.startswith(('cmp','test')) else put(xs[1],v,w)
 if op=='negl' and not flag_after[a]:return put(xs[0],'(0u-'+get(xs[0])+')')
 if op=='negl':return put(xs[0],f'alu(s,0,{get(xs[0])},32,1)')
 if op=='notl':return put(xs[0],'~'+get(xs[0]))
 if op in ('sarl','shrl','shll'):
  n=get(xs[0]) if len(xs)==2 else '1u'; dst=xs[-1]
  if not flag_after[a]:
   value=get(dst); value=f'(int32_t){value}' if op=='sarl' else value
   return put(dst,f'(uint32_t)({value}{"<<" if op=="shll" else ">>"}({n}&31u))')
  return put(dst,f'shift(s,{get(dst)},{n},{dict(sarl=0,shrl=1,shll=2)[op]})')
 if op=='shrdl':return put(xs[2],f'shrd(s,{get(xs[2])},{get(xs[1])},{get(xs[0])})')
 if op=='cltd':return 'edx=(uint32_t)((int32_t)eax>>31);'
 if op=='imull':
  if len(xs)==1:return f'q=(uint64_t)((int64_t)(int32_t)eax*(int32_t){get(xs[0])}); eax=(uint32_t)q; edx=q>>32;'
  return put(xs[-1],f'(uint32_t)((int64_t)(int32_t){get(xs[-1] if len(xs)==2 else xs[1])}*(int32_t){get(xs[0])})')
 if op=='mull':return f'q=(uint64_t)eax*{get(xs[0])}; eax=(uint32_t)q; edx=q>>32;'
 if op in ('divl','idivl'):
  return f'divide(s,{get(xs[0])},{1 if op=="idivl" else 0});'
 raise ValueError((hex(a),op,args))
b=dll.read_bytes(); pe=struct.unpack_from('<I',b,60)[0]; nsects=struct.unpack_from('<H',b,pe+6)[0]; optsize=struct.unpack_from('<H',b,pe+20)[0]
data=bytearray(0x15000)
for i in range(nsects):
 off=pe+24+optsize+i*40; name,vs,va,sz,raw=struct.unpack_from('<8sIIII',b,off)
 if va>=0x24000 and va<0x39000:data[va-0x24000:va-0x24000+sz]=b[raw:raw+sz]
(p/'mpz-tables.inc').write_text('static const uint8_t tables[0x15000]={\n'+''.join(','.join(str(v) for v in data[i:i+32])+',\n' for i in range(0,len(data),32))+'};\n')
out=['/* Code generated by lift.py; DO NOT EDIT. See README.md for provenance. */', '#include "mpz-native.h"']
for f in sorted(funcs):out.append(f'static void f_{f:x}(State *s);')
out.append('static void callback(State *s,uint32_t target){switch(target){')
for f in (0x10018e24,0x10018e10):out.append(f'case 0x{f:x}: f_{f:x}(s); return;')
out.append('default: hostcall(s,target); }}')
for f,seen in sorted(funcs.items()):
 out.append(f'static void f_{f:x}(State *s){{uint32_t t; uint64_t q; goto L{f:x};')
 for a in sorted(seen):
  op,args=ins[a]; out.append(f'L{a:x}: /* {op} {args} */ TRACE(s,0x{a:x}); '+emit(a,op,args))
  if op not in ('retl','jmp') and nxt[a] not in seen:raise ValueError('missing next')
 out.append('}')
(p/'decoder.c').write_text('\n'.join(out)+'\n#include "mpz-entry.inc"\n')
