SELECT
    rowid,
    config, 
    fail,
    pass
FROM TestResults
WHERE fail > 0
  AND pass = 0
  AND json_array_length(json_extract(config, '$.drops')) == 0
  AND json_array_length(json_extract(config, '$.corruptions')) == 1
;