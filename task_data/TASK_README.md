# System prototype
The prototype must be able to work with a configuration file and a set of external events of a certain format.
Solution should contain golang (1.22 or newer) source file/files and unit tests (optional)


## Description
A player is participating in a challenge. The goal is to completely clear a dungeon. The player navigates through floors and fights monsters. We need to process the events and compile the information into a final report


## Rules
1.  Only registered players are allowed to participate in the challenge
2.  The challenge ends if:
    1.  The player leaves the dungeon
    2.  The player cannot continue the challenge
    3.  The dungeon opening time has expired
    4.  Player is dead (health drops to 0)   
3.  When entering the boss's floor, the player receives a notification
4.  The boss floor does not contain any monsters
5.  The dungeon is considered complete if:
    1.  All floors are cleared of monsters
    2.  The boss is defeated
6.  A floor is considered complete when all monsters or the boss have been killed; ***any time spent in that floor is no longer counted***
7.  The player's health cannot exceed 100


## Events
- All events occur sequentially in time. (***Time of event N+1***) >= (***Time of event N***)
- Time format ***[HH:MM:SS]***. Trailing zeros are required in input and output
- The ***ExtraParam*** parameter can be a string containing multiple words.

#### Incoming events
| EventID   | ExtraParam    | Comment			                                  |
| ----------|:-------------:|-----------------------------------------------------|
|   1 		|				|   Player [`id`] registered				          |
|   2	    |				|	Player [`id`] entered the dungeon                 |
|   3	    |				|	Player [`id`] killed the monster                  |
|   4	    |				|	Player [`id`] went to the next floor              |
|   5	    |				|	Player [`id`] went to the previous floor     	  |
|   6	    |				|	Player [`id`] entered the boss's floor            |
|   7	    |				| 	Player [`id`] killed the boss                     |
|   8		|				|	Player [`id`] left the dungeon                    |
|   9		|	`reason`	|	Player [`id`] cannot continue due to [`reason`]   |
|   10		|	`health`	|	Player [`id`] has restored [`health`] of health   |
|   11		|	`damage`	|	Player [`id`] recieved [`damage`] of damage       |

#### Outgoing events
| EventID   | ExtraParam    | Comment			                                |
| ----------|:-------------:|---------------------------------------------------|
|   31 		|				|   Player [`id`] disqualified                      |
|   32	    |				|	Player [`id`] is dead                           |
|   33	    |				|	Player [`id`] makes imposible move [`eventID`]  |

#### Example

```
[14:00:00] 1 1
[14:00:00] 2 1
[14:10:00] 2 2
[14:10:00] 3 2
[14:11:00] 2 5
[14:12:00] 3 3
[14:14:00] 2 3
[14:27:00] 2 11 60
[14:29:00] 2 11 50
[14:40:00] 1 2
[14:41:00] 1 3
[14:44:00] 1 11 50
[14:45:00] 1 3
[14:48:00] 1 4
[14:48:00] 1 6
[14:49:00] 1 11 25
[14:49:02] 1 10 80
[14:50:00] 1 11 65
[14:59:00] 1 7
[15:04:00] 1 8
```

#### Output
```
[14:00:00] Player [1] registered
[14:00:00] Player [2] registered
[14:10:00] Player [2] entered the dungeon
[14:10:00] Player [3] is disqualified
[14:11:00] Player [2] makes imposible move [5]
[14:14:00] Player [2] killed the monster
[14:27:00] Player [2] recieved [60] of damage
[14:29:00] Player [2] recieved [50] of damage
[14:29:00] Player [2] is dead
[14:40:00] Player [1] entered the dungeon
[14:41:00] Player [1] killed the monster
[14:44:00] Player [1] recieved [50] of damage
[14:45:00] Player [1] killed the monster
[14:48:00] Player [1] went to the next floor
[14:48:00] Player [1] entered the boss's floor
[14:49:00] Player [1] recieved [25] of damage
[14:49:02] Player [1] has restored [80] of health
[14:50:00] Player [1] recieved [65] of damage
[14:59:00] Player [1] killed the boss
[15:04:00] Player [1] left the dungeon
```

## Configuration (json)
- **Floors**        - Number of floors in the dungeon
- **Monsters**      - Number of monsters on each floor of the dungeon
- **OpenAt**        - Dungeon opening time
- **Duration**      - Time until the dungeon closes in hours


#### Example
```
{
    "Floors": 2,
    "Monsters": 2,
    "OpenAt": "14:05:00",
    "Duration": 2
}
```

## States
| State     | Comment                                                           |
| ----------|:-----------------------------------------------------------------:|
|   SUCCESS |   All floors are cleared                                          |
|   FAIL	|	The player died or the dungeon is not considered completed      |
|   DISQUAL	|   The player cannot continue or has not completed registration    |

## Final report
1.  State `SUCCESS`/`FAIL`/`DISQUAL`
2.  Player ID
3.  Time spent in the dungeon (all the time until the player left the dungeon or the dungeon closed)
4.  Average time to clear a floor of monsters (the boss's floor is not included in the calculation)
5.  Time to kill the boss
6.  Player health at the end of the trial

#### Example
```
Final report:
[SUCCESS] 1 [00:24:00, 00:05:00, 00:11:00] HP:35
[FAIL] 2 [00:19:00, 00:00:00, 00:00:00] HP:0
[DISQUAL] 3 [00:00:00, 00:00:00, 00:00:00] HP:100
```