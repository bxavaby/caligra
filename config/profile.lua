function Randomize(base)
    local variations = {
        ["nynynn"] = { "nynynn", "nyvynn", "nynyn" },
    }

    if variations[base] then
        local options = variations[base]
        return options[math.random(#options)]
    end
    return base
end

return {
    author = Randomize("nynynn"),
    software = "liberty/1.0",
    created = "2000-05-01",
    organization = "Untraceable / Decentralized",
    location = "Stateless",
    comment = "Restructure everything",
    profile_version = "1.0"
}
