So hear me out

instead im pushing the boundary if discord embed , im gonna scale down. so instead detailed stats im just showing their respective skills i.e from support hint or event skills

so we leveraging the {build_id}/support.json as core endpoint , and use the {slug_id} for populating the image later on

when user invoke the command either using name or the full slug/url_name we return what available skills that support card had

from what we have right now the gametora_client.go

we simplify the details needed for embed , right now the plan was to cram every single details inside embed which quite complex since its pushing the discord embed to the limit without hack around

we keep the newgametoraclient , get build id function , search support card , get support card image and revamp the caching mechanism

here the core endpoint

https://gametora.com/_next/data/4Lod4e9rq2HCjy-VKjMHJ/umamusume/supports.json ( this one is the core endpoint consist of every single skill data etc, check the sample.json ) 

https://gametora.com/_next/data/4Lod4e9rq2HCjy-VKjMHJ/umamusume/characters/daring-tact.json?id=daring-tact ( this one for fetching the card metadata or check the metadata.json ) or https://gametora.com/_next/data/Yc0z5VqNnvNNtiPlBZavp/umamusume/supports/30261-win-variation.json?id=30261-win-variation ( totally the same return res )

https://gametora.com/images/umamusume/supports/tex_support_card_{slug_id}.png ( this one for the image grabber based on the slug id ) or https://gametora.com/images/umamusume/supports/tex_support_card_10001.png ( the example from the website )

now for the proposed embed struct

right now its defined as such

スペシャルウィーク (スペシャルウィーク)
Character: Special Week (Special Week)
🎴 Rarity
1
🎯 Type
guts
🆔 Card ID
🎉 Events (1)
Backup Belly (Event Skill 200762)
• Type: [lng], Rarity: 1
• Choices:
Description not available (Description not available)
Description not available (Description not available)
✨ Effects (7)
Level 1 (Level 1)
• Effect: 5
• Type: Level 1

Level 2 (Level 2)
• Effect: 10
• Type: Level 2

Level 5 (Level 5)
• Effect: -1
• Type: Level 5

Level 14 (Level 14)
• Effect: -1
• Type: Level 14

Level 17 (Level 17)
• Effect: -1
• Type: Level 17

Level 18 (Level 18)
• Effect: -1
• Type: Level 18

Level 19 (Level 19)
• Effect: -1
• Type: Level 19
💡 Hints (10)
Bad Track Condition ○ (Skill 200162)
• Type: [nac]

Rainy Days ○ (Skill 200232)
• Type: [nac]

Last Leg (Skill 200512)
• Type: [l_3 nac]

Outside Pass, Ready! (Skill 200612)
• Type: [btw f_c cor]

Stand Your Ground (Skill 200732)
• Type: [med f_s f_c]

Nutritional Supplements (Skill 201352)
• Type: [ldr l_1]

Betweener's Tricks ○ (Skill 201542)
• Type: [btw]

Hint Type 1 (Hint Type 1)
• Value: 1

Hint Type 3 (Hint Type 3)
• Value: 1

Hint Type 4 (Hint Type 4)
• Value: 6

[image]

footer

its decent but cluttered with garbage info / not populated properly

so here the proposed struct

スペシャルウィーク

Character: Special Week

🎴 Rarity  🎯 Type  🆔 Support ID  Obtained ( optional if the embed inline support )
1           guts       10001       {"obtained"}

Support Hint

"name_en": "Bad Track Condition ○",

Event Skill

"name_en": "Backup Belly",

