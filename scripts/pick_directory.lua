-- Neovim Lua script for directory picker using Oil.nvim
-- This script opens Oil.nvim for directory selection and writes the selected path to a temp file

-- Ensure Oil is available
local ok, oil = pcall(require, "oil")
if not ok then
    print("Error: Oil.nvim is not installed or not available")
    vim.cmd("q!")
    return
end

-- File to write the selected directory path
local output_file = vim.fn.argv(0) or "/tmp/claude_squad_selected_dir"

-- Function to check if directory is a git repository
local function is_git_repo(path)
    local git_dir = path .. "/.git"
    local stat = vim.loop.fs_stat(git_dir)
    return stat ~= nil
end

-- Function to select current directory
local function select_current_directory()
    local current_dir = oil.get_current_dir()
    if current_dir then
        -- Remove trailing slash if present
        current_dir = current_dir:gsub("/$", "")
        
        -- Check if it's a git repository
        if not is_git_repo(current_dir) then
            vim.notify("Error: Selected directory is not a git repository", vim.log.levels.ERROR)
            return
        end
        
        -- Write the selected directory to output file
        local file = io.open(output_file, "w")
        if file then
            file:write(current_dir)
            file:close()
            vim.notify("Selected: " .. current_dir, vim.log.levels.INFO)
            vim.cmd("q!")
        else
            vim.notify("Error: Could not write to output file", vim.log.levels.ERROR)
        end
    else
        vim.notify("Error: Could not determine current directory", vim.log.levels.ERROR)
    end
end

-- Function to cancel selection
local function cancel_selection()
    -- Write empty string to indicate cancellation
    local file = io.open(output_file, "w")
    if file then
        file:write("")
        file:close()
    end
    vim.cmd("q!")
end

-- Configure Oil for directory selection
oil.setup({
    default_file_explorer = true,
    columns = {
        "icon",
        "permissions",
        "size",
        "mtime",
    },
    buf_options = {
        buflisted = false,
        bufhidden = "hide",
    },
    win_options = {
        wrap = false,
        signcolumn = "no",
        cursorcolumn = false,
        foldcolumn = "0",
        spell = false,
        list = false,
        conceallevel = 3,
        concealcursor = "nvic",
    },
    delete_to_trash = false,
    skip_confirm_for_simple_edits = false,
    prompt_save_on_select_new_entry = true,
    cleanup_delay_ms = 2000,
    lsp_file_methods = {
        timeout_ms = 1000,
        autosave_changes = false,
    },
    constrain_cursor = "editable",
    experimental_watch_for_changes = false,
    keymaps = {
        ["g?"] = "actions.show_help",
        ["<CR>"] = "actions.select",
        ["<C-s>"] = "actions.select_vsplit",
        ["<C-h>"] = "actions.select_split",
        ["<C-t>"] = "actions.select_tab",
        ["<C-p>"] = "actions.preview",
        ["<C-c>"] = "actions.close",
        ["<C-l>"] = "actions.refresh",
        ["-"] = "actions.parent",
        ["_"] = "actions.open_cwd",
        ["`"] = "actions.cd",
        ["~"] = "actions.tcd",
        ["gs"] = "actions.change_sort",
        ["gx"] = "actions.open_external",
        ["g."] = "actions.toggle_hidden",
        ["g\\"] = "actions.toggle_trash",
        -- Custom keymaps for directory selection
        ["<space>"] = select_current_directory,
        ["S"] = select_current_directory,
    },
    use_default_keymaps = true,
    view_options = {
        show_hidden = false,
        is_hidden_file = function(name, bufnr)
            return vim.startswith(name, ".")
        end,
        is_always_hidden = function(name, bufnr)
            return false
        end,
        sort = {
            { "type", "asc" },
            { "name", "asc" },
        },
    },
})

-- Set up additional keymaps that definitely override any defaults
vim.keymap.set("n", "<Esc>", cancel_selection, { 
    buffer = true, 
    desc = "Cancel directory selection" 
})

vim.keymap.set("n", "q", cancel_selection, { 
    buffer = true, 
    desc = "Cancel directory selection" 
})

-- Show help message
vim.notify("Claude Squad Directory Picker", vim.log.levels.INFO)
vim.notify("Navigate: j/k (up/down), Enter (open dir), - (parent dir)", vim.log.levels.INFO)
vim.notify("Select: 'S' or Space (select current directory), Esc/q (cancel)", vim.log.levels.INFO)

-- Start in home directory
local home_dir = vim.fn.expand("~")
vim.cmd("Oil " .. home_dir)