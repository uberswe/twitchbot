let data = {hash: document.location.hash};
fetch("/auth", {
    method: "POST",
    body: JSON.stringify(data)
}).then(res => {
    window.location = "/admin"
}).catch(res => {
    console.log("an error occured", res)
});