package craft

var serverPropertiesFileName string = "server.properties" // Name of the server properties (configuration) file
var serverPropertiesDefaultBody string = `server-name=Dedicated Server
# Used as the server name
# Allowed values: Any string

gamemode=survival
# Sets the game mode for new players.
# Allowed values: "survival", "creative", or "adventure"

difficulty=easy
# Sets the difficulty of the world.
# Allowed values: "peaceful", "easy", "normal", or "hard"

allow-cheats=false
# If true then cheats like commands can be used.
# Allowed values: "true" or "false"

max-players=10
# The maximum number of players that can play on the server.
# Allowed values: Any positive integer

online-mode=true
# If true then all connected players must be authenticated to Xbox Live.
# Clients connecting to remote (non-LAN) servers will always require Xbox Live authentication regardless of this setting.
# If the server accepts connections from the Internet, then it's highly recommended to enable online-mode.
# Allowed values: "true" or "false"

white-list=false
# If true then all connected players must be listed in the separate whitelist.json file.
# Allowed values: "true" or "false"

server-port=19132
# Which IPv4 port the server should listen to.
# Allowed values: Integers in the range [1, 65535]

server-portv6=19133
# Which IPv6 port the server should listen to.
# Allowed values: Integers in the range [1, 65535]

view-distance=32
# The maximum allowed view distance in number of chunks.
# Allowed values: Any positive integer.

tick-distance=4
# The world will be ticked this many chunks away from any player.
# Allowed values: Integers in the range [4, 12]

player-idle-timeout=30
# After a player has idled for this many minutes they will be kicked. If set to 0 then players can idle indefinitely.
# Allowed values: Any non-negative integer.

max-threads=8
# Maximum number of threads the server will try to use. If set to 0 or removed then it will use as many as possible.
# Allowed values: Any positive integer.

level-name=Bedrock level
# Allowed values: Any string

level-seed=
# Use to randomize the world
# Allowed values: Any string

default-player-permission-level=member
# Permission level for new players joining for the first time.
# Allowed values: "visitor", "member", "operator"

texturepack-required=false
# Force clients to use texture packs in the current world
# Allowed values: "true" or "false"

content-log-file-enabled=false
# Enables logging content errors to a file
# Allowed values: "true" or "false"

compression-threshold=1
# Determines the smallest size of raw network payload to compress
# Allowed values: 0-65535

server-authoritative-movement=server-auth
# Allowed values: "client-auth", "server-auth"
# Enables server authoritative movement. If "server-auth", the server will replay local user input on
# the server and send down corrections when the client's position doesn't match the server's.
# Corrections will only happen if correct-player-movement is set to true.

player-movement-score-threshold=20
# The number of incongruent time intervals needed before abnormal behavior is reported.
# Disabled by server-authoritative-movement.

player-movement-distance-threshold=0.3
# The difference between server and client positions that needs to be exceeded before abnormal behavior is detected.
# Disabled by server-authoritative-movement.

player-movement-duration-threshold-in-ms=500
# The duration of time the server and client positions can be out of sync (as defined by player-movement-distance-threshold)
# before the abnormal movement score is incremented. This value is defined in milliseconds.
# Disabled by server-authoritative-movement.

correct-player-movement=false
# If true, the client position will get corrected to the server position if the movement score exceeds the threshold.
`
