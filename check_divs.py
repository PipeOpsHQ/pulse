import sys

content = open('frontend/src/pages/ProjectDetail.svelte').read()
level = 0
if_level = 0
lines = content.split('\n')
for i, line in enumerate(lines):
    # Skip comments
    if '<!--' in line and '-->' in line:
        line = line.split('<!--')[0] + line.split('-->')[1]
    
    opens = line.count('<div')
    closes = line.count('</div')
    
    if_opens = line.count('{#if') + line.count('{#each') + line.count('{#await') + line.count('{#snippet')
    if_closes = line.count('{/if}') + line.count('{/each}') + line.count('{/await}') + line.count('{/snippet}')
    
    if opens > 0 or closes > 0 or if_opens > 0 or if_closes > 0:
        level += opens - closes
        if_level += if_opens - if_closes
        if level < 0 or if_level < 0:
             print(f"ERROR at line {i+1}: div_level={level}, if_level={if_level} | {line.strip()}")
        # print(f"{i+1:4}: div={level:2} if={if_level:2} | {line.strip()}")

print(f"Final levels: div={level}, if={if_level}")
