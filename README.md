# Module viamtriggers 


## movement-motion

Uses a movement sensor to detect motion, and turn a switch on or off

```json
{
   "sensor" : <name>, // required, the movement sensor
   "switch" : <name>, // required, name of the sensor
   "position" : { "motion" : 1, "idle" : 0 }, // optional, what position to set when a detection happes
   "idle-minutes" : <0>, // optional, number of minutes to leave switch on/or off, 0 means forever
   "threshold" : 10, // optional, threshold for angularvelocity movement
}
```

## sunset-lights

Turn lights on after sunset

```json
{
   "switch" : <name>, // required, name of the sensor
   "lat" : <latitude>, // required
   "lng" : <longitude> // required
}
```
