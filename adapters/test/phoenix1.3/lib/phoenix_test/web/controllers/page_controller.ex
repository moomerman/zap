defmodule PhoenixTest.Web.PageController do
  use PhoenixTest.Web, :controller

  def index(conn, _params) do
    render conn, "index.html"
  end
end
