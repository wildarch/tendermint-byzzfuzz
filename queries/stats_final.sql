SELECT
    json_array_length(json_extract(config, '$.drops')) AS drops,
    json_array_length(json_extract(config, '$.corruptions')) AS corruptions,

    COUNT(*) AS configs,
    SUM(CASE WHEN fail > 0 THEN 1 ELSE 0 END) AS fail,
    SUM(CASE WHEN fail > 1 THEN 1 ELSE 0 END) AS fail2,
    SUM(CASE WHEN fail >= 5 AND pass = 0 THEN 1 ELSE 0 END) AS fail_reliable,
    SUM(CASE WHEN pass > 0 THEN 1 ELSE 0 END) AS pass,
    SUM(CASE WHEN fail > 0 AND pass > 0 THEN 1 ELSE 0 END) AS flaky

FROM TestResults
WHERE is_final
GROUP BY 1, 2
ORDER BY 1, 2;